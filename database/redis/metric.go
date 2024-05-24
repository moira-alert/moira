package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/database/redis/reply"
	"github.com/patrickmn/go-cache"
	"gopkg.in/tomb.v2"
)

var (
	ErrCleanUpDurationLessThanZero    = errors.New("clean up duration value must be greater than zero, otherwise the current metrics may be deleted")
	ErrCleanUpDurationGreaterThanZero = errors.New("clean up duration value must be less than zero, otherwise all metrics will be removed")
)

func (connector *DbConnector) addPatterns(patterns ...string) error {
	ctx := connector.context
	client := *connector.client

	if _, err := client.SAdd(ctx, patternsListKey, patterns).Result(); err != nil {
		return fmt.Errorf("failed to add moira patterns, error: %w", err)
	}

	return nil
}

// GetPatterns gets updated patterns array.
func (connector *DbConnector) GetPatterns() ([]string, error) {
	ctx := connector.context
	client := *connector.client

	patterns, err := client.SMembers(ctx, patternsListKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get moira patterns, error: %w", err)
	}

	return patterns, nil
}

// GetMetricsValues gets metrics values for given interval.
func (connector *DbConnector) GetMetricsValues(metrics []string, from int64, until int64) (map[string][]*moira.MetricValue, error) {
	c := *connector.client
	ctx := connector.context

	pipe := c.TxPipeline()

	for _, metric := range metrics {
		rng := &redis.ZRangeBy{
			Min: strconv.FormatInt(from, 10),
			Max: strconv.FormatInt(until, 10),
		}
		pipe.ZRangeByScoreWithScores(connector.context, metricDataKey(metric), rng)
	}

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to Exec in get metrics values: %w", err)
	}

	resultByMetrics := make([]*redis.ZSliceCmd, 0, len(metrics))

	for _, cmd := range cmds {
		res, ok := cmd.(*redis.ZSliceCmd)
		if !ok {
			return nil, fmt.Errorf("failed to convert cmd response to *ZSliceCmd in get metrics values")
		}

		resultByMetrics = append(resultByMetrics, res)
	}

	result := make(map[string][]*moira.MetricValue, len(resultByMetrics))

	for i, resultByMetric := range resultByMetrics {
		metric := metrics[i]
		metricsValues, err := reply.MetricValues(resultByMetric)
		if err != nil {
			return nil, fmt.Errorf("failed to convert ZSliceCmd to metric values in get metrics values: %w", err)
		}

		result[metric] = metricsValues
	}

	return result, nil
}

// GetMetricRetention gets given metric retention, if retention is empty then return default retention value(60).
func (connector *DbConnector) GetMetricRetention(metric string) (int64, error) {
	retention, ok := connector.getCachedRetention(metric)
	if ok {
		return retention, nil
	}
	retention, err := connector.getMetricRetention(metric)
	if err != nil {
		if errors.Is(err, database.ErrNil) {
			return retention, nil
		}
		return retention, err
	}
	connector.retentionCache.Set(metric, retention, 0)
	return retention, nil
}

func (connector *DbConnector) getCachedRetention(metric string) (int64, bool) {
	value, ok := connector.retentionCache.Get(metric)
	if !ok {
		return 0, false
	}
	retention, ok := value.(int64)
	return retention, ok
}

func (connector *DbConnector) getMetricRetention(metric string) (int64, error) {
	c := *connector.client

	retentionStr, err := c.Get(connector.context, metricRetentionKey(metric)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 60, database.ErrNil //nolint
		}
		return 0, fmt.Errorf("failed GET metric retention:%s, error: %w", metric, err)
	}
	retention, err := strconv.ParseInt(retentionStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed GET metric retention:%s, error: %w", metric, err)
	}
	return retention, nil
}

// SaveMetrics saves new metrics.
func (connector *DbConnector) SaveMetrics(metrics map[string]*moira.MatchedMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	var err error
	c := *connector.client
	ctx := connector.context

	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	pipe := c.TxPipeline()

	for _, metric := range metrics {
		metricValue := fmt.Sprintf("%v %v", metric.Timestamp, metric.Value)
		z := &redis.Z{Score: float64(metric.RetentionTimestamp), Member: metricValue}
		if err = c.ZAdd(ctx, metricDataKey(metric.Metric), z).Err(); err != nil {
			return err
		}

		if err = connector.retentionSavingCache.Add(metric.Metric, true, cache.DefaultExpiration); err == nil {
			if err = c.Set(ctx, metricRetentionKey(metric.Metric), metric.Retention, redis.KeepTTL).Err(); err != nil {
				return err
			}
		}

		for _, pattern := range metric.Patterns {
			if err = c.SAdd(ctx, patternMetricsKey(pattern), metric.Metric).Err(); err != nil {
				return err
			}

			var event []byte
			event, err = json.Marshal(&moira.MetricEvent{
				Metric:  metric.Metric,
				Pattern: pattern,
			})
			if err != nil {
				continue
			}

			metricEventsChannel := metricEventsChannels[rand.Intn(len(metricEventsChannels))]
			pipe.SAdd(ctx, metricEventsChannel, event)
		}
	}

	if _, err = pipe.Exec(ctx); err != nil {
		connector.logger.Error().
			Error(err).
			Msg("Sending metric event error")
		return err
	}

	return nil
}

// SubscribeMetricEvents creates subscription for new metrics and return channel for this events.
func (connector *DbConnector) SubscribeMetricEvents(tomb *tomb.Tomb, params *moira.SubscribeMetricEventsParams) (<-chan *moira.MetricEvent, error) {
	responseChannel := make(chan string, metricEventChannelSize)
	metricChannel := make(chan *moira.MetricEvent, metricEventChannelSize)

	ctx := connector.context
	c := *connector.client

	if err := c.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	go func() {
		for {
			response, ok := <-responseChannel
			if !ok {
				close(metricChannel)
				return
			}

			metricEvent := &moira.MetricEvent{}
			if err := json.Unmarshal([]byte(response), metricEvent); err != nil {
				connector.logger.Error().
					String("metric_event", response).
					Error(err).
					Msg("Failed to parse MetricEvent")
				continue
			}
			metricChannel <- metricEvent
		}
	}()

	totalEventsChannels := len(metricEventsChannels)
	closedEventChannels := int32(0)

	for channelIdx := 0; channelIdx < totalEventsChannels; channelIdx++ {
		metricEventsChannel := metricEventsChannels[channelIdx]
		go func() {
			var popDelay time.Duration
			for {
				startPop := time.After(popDelay)
				select {
				case <-tomb.Dying():
					if atomic.AddInt32(&closedEventChannels, 1) == int32(totalEventsChannels) {
						close(responseChannel)
					}

					return
				case <-startPop:
					data, err := c.SPopN(ctx, metricEventsChannel, params.BatchSize).Result()
					popDelay = connector.handlePopResponse(data, err, responseChannel, params.Delay)
				}
			}
		}()
	}

	return metricChannel, nil
}

const (
	receiveErrorSleepDuration = time.Second
	receiveEmptySleepDuration = time.Second
)

func (connector *DbConnector) handlePopResponse(data []string, popError error, responseChannel chan string, defaultDelay time.Duration) time.Duration {
	if popError != nil {
		if !errors.Is(popError, redis.Nil) {
			connector.logger.Error().
				Error(popError).
				Msg("Failed to pop new metric events")
		}
		return receiveErrorSleepDuration
	} else if len(data) == 0 {
		return receiveEmptySleepDuration
	}

	for _, response := range data {
		responseChannel <- response
	}

	return defaultDelay
}

// AddPatternMetric adds new metrics by given pattern.
func (connector *DbConnector) AddPatternMetric(pattern, metric string) error {
	c := *connector.client
	if _, err := c.SAdd(connector.context, patternMetricsKey(pattern), metric).Result(); err != nil {
		return fmt.Errorf("failed to SADD pattern-metrics, pattern: %s, metric: %s, error: %w", pattern, metric, err)
	}

	return nil
}

// GetPatternMetrics gets all metrics by given pattern.
func (connector *DbConnector) GetPatternMetrics(pattern string) ([]string, error) {
	c := *connector.client

	metrics, err := c.SMembers(connector.context, patternMetricsKey(pattern)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return make([]string, 0), nil
		}
		return nil, fmt.Errorf("failed to get pattern metrics for pattern %s, error: %w", pattern, err)
	}
	return metrics, nil
}

// RemovePattern removes pattern from patterns list.
func (connector *DbConnector) RemovePattern(pattern string) error {
	c := *connector.client
	if _, err := c.SRem(connector.context, patternsListKey, pattern).Result(); err != nil {
		return fmt.Errorf("failed to remove pattern: %s, error: %w", pattern, err)
	}
	return nil
}

// RemovePatternsMetrics removes metrics by given patterns.
func (connector *DbConnector) RemovePatternsMetrics(patterns []string) error {
	pipe := (*connector.client).TxPipeline()
	for _, pattern := range patterns {
		pipe.Del(connector.context, patternMetricsKey(pattern)) //nolint
	}
	if _, err := pipe.Exec(connector.context); err != nil {
		return fmt.Errorf("failed to EXEC: %w", err)
	}
	return nil
}

// RemovePatternWithMetrics removes pattern metrics with data and given pattern.
func (connector *DbConnector) RemovePatternWithMetrics(pattern string) error {
	metrics, err := connector.GetPatternMetrics(pattern)
	if err != nil {
		return err
	}
	pipe := (*connector.client).TxPipeline()
	pipe.SRem(connector.context, patternsListKey, pattern)
	for _, metric := range metrics {
		pipe.Del(connector.context, metricDataKey(metric))
		pipe.Del(connector.context, metricRetentionKey(metric))
	}
	pipe.Del(connector.context, patternMetricsKey(pattern))
	if _, err = pipe.Exec(connector.context); err != nil {
		return fmt.Errorf("failed to EXEC: %w", err)
	}
	return nil
}

// RemoveMetricRetention remove metric retention.
func (connector *DbConnector) RemoveMetricRetention(metric string) error {
	c := *connector.client
	if _, err := c.Del(connector.context, metricRetentionKey(metric)).Result(); err != nil {
		return fmt.Errorf("failed to remove retention, error: %w", err)
	}

	return nil
}

// RemoveMetricValues remove values by metrics from the interval of passed parameters, if they are not in the metricsCache.
// In from and to, expect either -inf, +inf, or timestamps as strings.
func (connector *DbConnector) RemoveMetricValues(metric string, from, to string) (int64, error) {
	if !connector.needRemoveMetrics(metric) {
		return 0, nil
	}

	c := *connector.client
	result, err := c.ZRemRangeByScore(connector.context, metricDataKey(metric), from, to).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to remove metrics from %s to %s, error: %w", from, to, err)
	}

	return result, nil
}

// GetMetricsTTLSeconds returns maximum time in seconds to store metrics in Redis.
func (connector *DbConnector) GetMetricsTTLSeconds() int64 {
	return connector.metricsTTLSeconds
}

// RemoveMetricsValues remove metrics timestamps values from 0 to given time.
func (connector *DbConnector) RemoveMetricsValues(metrics []string, toTime int64) error {
	pipe := (*connector.client).TxPipeline()
	for _, metric := range metrics {
		if connector.needRemoveMetrics(metric) {
			pipe.ZRemRangeByScore(connector.context, metricDataKey(metric), "-inf", strconv.FormatInt(toTime, 10)) //nolint
		}
	}
	if _, err := pipe.Exec(connector.context); err != nil {
		return fmt.Errorf("failed to EXEC remove metrics: %w", err)
	}
	return nil
}

func (connector *DbConnector) needRemoveMetrics(metric string) bool {
	err := connector.metricsCache.Add(metric, true, 0)
	return err == nil
}

func cleanUpMetricsOnRedisNode(connector *DbConnector, client redis.UniversalClient, from, to string) error {
	metricsIterator := client.ScanType(connector.context, 0, metricDataKey("*"), 0, "zset").Iterator()
	var count int64

	for metricsIterator.Next(connector.context) {
		key := metricsIterator.Val()
		metric := strings.TrimPrefix(key, metricDataKey(""))

		deletedCount, err := connector.RemoveMetricValues(metric, from, to)
		if err != nil {
			return err
		}

		count += deletedCount
	}

	connector.logger.Info().
		Int64("count deleted metrics", count).
		Msg("Cleaned up metrics")

	return nil
}

func cleanUpAbandonedRetentionsOnRedisNode(connector *DbConnector, client redis.UniversalClient) error {
	iter := client.Scan(connector.context, 0, metricRetentionKey("*"), 0).Iterator()
	for iter.Next(connector.context) {
		key := iter.Val()
		metric := strings.TrimPrefix(key, metricRetentionKey(""))

		result, err := (*connector.client).Exists(connector.context, metricDataKey(metric)).Result()
		if err != nil {
			return fmt.Errorf("failed to check metric data existence, error: %w", err)
		}
		if isMetricExists := result == 1; !isMetricExists {
			err = connector.RemoveMetricRetention(metric)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (connector *DbConnector) CleanUpOutdatedMetrics(duration time.Duration) error {
	if duration >= 0 {
		return ErrCleanUpDurationGreaterThanZero
	}

	from := "-inf"
	toTs := time.Now().UTC().Add(duration).Unix()
	to := strconv.FormatInt(toTs, 10)

	return connector.callFunc(func(connector *DbConnector, client redis.UniversalClient) error {
		return cleanUpMetricsOnRedisNode(connector, client, from, to)
	})
}

func (connector *DbConnector) CleanUpFutureMetrics(duration time.Duration) error {
	if duration <= 0 {
		return ErrCleanUpDurationLessThanZero
	}

	fromTs := connector.clock.Now().Add(duration).Unix()
	from := strconv.FormatInt(fromTs, 10)
	to := "+inf"

	return connector.callFunc(func(connector *DbConnector, client redis.UniversalClient) error {
		return cleanUpMetricsOnRedisNode(connector, client, from, to)
	})
}

// CleanupOutdatedPatternMetrics removes already deleted metrics from the moira-pattern-metrics key.
func (connector *DbConnector) CleanupOutdatedPatternMetrics() (int64, error) {
	var count int64

	ctx := connector.context
	client := *connector.client

	patterns, err := connector.GetPatterns()
	if err != nil {
		return count, fmt.Errorf("failed to get patterns: %w", err)
	}

	pipe := client.TxPipeline()

	for _, pattern := range patterns {
		nonExistentMetrics, err := connector.getNonExistentPatternMetrics(pattern)
		if err != nil {
			return count, fmt.Errorf("failed to get non existent metrics by pattern: %w", err)
		}

		for _, metric := range nonExistentMetrics {
			pipe.SRem(ctx, patternMetricsKey(pattern), metric)
			count++
		}
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return count, fmt.Errorf("failed to remove outdated pattern metrics: %w", err)
	}

	return count, nil
}

func (connector *DbConnector) getNonExistentPatternMetrics(pattern string) ([]string, error) {
	ctx := connector.context
	client := *connector.client

	metrics, err := connector.GetPatternMetrics(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get pattern metrics: %w", err)
	}

	pipe := client.TxPipeline()

	for _, metric := range metrics {
		pipe.Exists(ctx, metricDataKey(metric))
	}

	exec, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to Exec Exists metric by pattern: %w", err)
	}

	nonExistentMetrics := make([]string, 0)

	for i, cmder := range exec {
		cmd, ok := cmder.(*redis.IntCmd)
		if !ok {
			return nil, fmt.Errorf("failed to convert cmder to intcmd result: %w", err)
		}

		res, err := cmd.Result()
		if err != nil {
			return nil, err
		}

		if res == 0 {
			nonExistentMetrics = append(nonExistentMetrics, metrics[i])
		}
	}

	return nonExistentMetrics, nil
}

// CleanUpAbandonedRetentions removes metric retention keys that have no corresponding metric data.
func (connector *DbConnector) CleanUpAbandonedRetentions() error {
	return connector.callFunc(cleanUpAbandonedRetentionsOnRedisNode)
}

func removeMetricsByPrefixOnRedisNode(connector *DbConnector, client redis.UniversalClient, prefix string) error {
	metricDataIterator := client.Scan(connector.context, 0, metricDataKey(fmt.Sprintf("%s*", prefix)), 0).Iterator()
	for metricDataIterator.Next(connector.context) {
		err := client.Del(connector.context, metricDataIterator.Val()).Err()
		if err != nil {
			return err
		}
	}

	metricRetentionIterator := client.Scan(connector.context, 0, metricRetentionKey(fmt.Sprintf("%s*", prefix)), 0).Iterator()
	for metricRetentionIterator.Next(connector.context) {
		err := client.Del(connector.context, metricRetentionIterator.Val()).Err()
		if err != nil {
			return err
		}
	}

	patternMetricsIterator := client.Scan(connector.context, 0, patternMetricsKey("*"), 0).Iterator()
	for patternMetricsIterator.Next(connector.context) {
		patternMetricsSetKey := patternMetricsIterator.Val()
		patternMetrics := client.SMembers(connector.context, patternMetricsSetKey).Val()
		for _, patternMetric := range patternMetrics {
			if strings.HasPrefix(patternMetric, prefix) {
				err := client.SRem(connector.context, patternMetricsSetKey, patternMetric).Err()
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// RemoveMetricsByPrefix removes metrics by their prefix e.g. "my.super.metric.".
func (connector *DbConnector) RemoveMetricsByPrefix(prefix string) error {
	return connector.callFunc(func(connector *DbConnector, client redis.UniversalClient) error {
		return removeMetricsByPrefixOnRedisNode(connector, client, prefix)
	})
}

func removeAllMetricsOnRedisNode(connector *DbConnector, client redis.UniversalClient) error {
	metricDataIterator := client.Scan(connector.context, 0, metricDataKey("*"), 0).Iterator()
	for metricDataIterator.Next(connector.context) {
		err := client.Del(connector.context, metricDataIterator.Val()).Err()
		if err != nil {
			return err
		}
	}

	metricRetentionIterator := client.Scan(connector.context, 0, metricRetentionKey("*"), 0).Iterator()
	for metricRetentionIterator.Next(connector.context) {
		err := client.Del(connector.context, metricRetentionIterator.Val()).Err()
		if err != nil {
			return err
		}
	}

	patternMetricsIterator := client.Scan(connector.context, 0, patternMetricsKey("*"), 0).Iterator()
	for patternMetricsIterator.Next(connector.context) {
		err := client.Del(connector.context, patternMetricsIterator.Val()).Err()
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveAllMetrics removes all metrics.
func (connector *DbConnector) RemoveAllMetrics() error {
	return connector.callFunc(removeAllMetricsOnRedisNode)
}

var patternsListKey = "moira-pattern-list"

var metricEventsChannels = []string{
	"metric-event-0",
	"metric-event-1",
	"metric-event-2",
	"metric-event-3",
}

func patternMetricsKey(pattern string) string {
	return "moira-pattern-metrics:" + pattern
}

func metricDataKey(metric string) string {
	return "moira-metric-data:" + metric
}

func metricRetentionKey(metric string) string {
	return "moira-metric-retention:" + metric
}
