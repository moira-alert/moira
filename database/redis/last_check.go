package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/moira-alert/moira/database"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis/reply"
)

// GetTriggerLastCheck gets trigger last check data by given triggerID, if no value, return database.ErrNil error
func (connector *DbConnector) GetTriggerLastCheck(triggerID string) (moira.CheckData, error) {
	ctx := connector.context
	c := *connector.client

	lastCheck, err := reply.Check(c.Get(ctx, metricLastCheckKey(triggerID)))
	if err != nil {
		return lastCheck, err
	}

	return lastCheck, nil
}

// SetTriggerLastCheck sets trigger last check data
func (connector *DbConnector) SetTriggerLastCheck(triggerID string, checkData *moira.CheckData, triggerSource moira.TriggerSource) error {
	selfStateCheckCountKey := connector.getSelfStateCheckCountKey(triggerSource)
	bytes, err := reply.GetCheckBytes(*checkData)
	if err != nil {
		return err
	}

	triggerNeedToReindex := connector.checkDataScoreChanged(triggerID, checkData)

	ctx := connector.context
	pipe := (*connector.client).TxPipeline()
	pipe.Set(ctx, metricLastCheckKey(triggerID), bytes, redis.KeepTTL)
	pipe.ZAdd(ctx, triggersChecksKey, &redis.Z{Score: float64(checkData.Score), Member: triggerID})

	if selfStateCheckCountKey != "" {
		pipe.Incr(ctx, selfStateCheckCountKey)
	}

	if checkData.Score > 0 {
		pipe.SAdd(ctx, badStateTriggersKey, triggerID)
	} else {
		pipe.SRem(ctx, badStateTriggersKey, triggerID)
	}

	if triggerNeedToReindex {
		pipe.ZAdd(ctx, triggersToReindexKey, &redis.Z{Score: float64(time.Now().Unix()), Member: triggerID})
	}

	_, err = pipe.Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	return nil
}

func (connector *DbConnector) getSelfStateCheckCountKey(triggerSource moira.TriggerSource) string {
	if connector.source != Checker {
		return ""
	}
	switch triggerSource {
	case moira.GraphiteLocal:
		return selfStateChecksCounterKey

	case moira.GraphiteRemote:
		return selfStateRemoteChecksCounterKey

	case moira.PrometheusRemote:
		return selfStatePrometheusChecksCounterKey

	default:
		return ""
	}
}

func appendRemoveTriggerLastCheckToRedisPipeline(ctx context.Context, pipe redis.Pipeliner, triggerID string) redis.Pipeliner {
	pipe.Del(ctx, metricLastCheckKey(triggerID))
	pipe.ZRem(ctx, triggersChecksKey, triggerID)
	pipe.SRem(ctx, badStateTriggersKey, triggerID)
	pipe.ZAdd(ctx, triggersToReindexKey, &redis.Z{Score: float64(time.Now().Unix()), Member: triggerID})

	return pipe
}

// RemoveTriggerLastCheck removes trigger last check data
func (connector *DbConnector) RemoveTriggerLastCheck(triggerID string) error {
	ctx := connector.context
	pipe := (*connector.client).TxPipeline()
	pipe = appendRemoveTriggerLastCheckToRedisPipeline(ctx, pipe, triggerID)
	_, err := pipe.Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	return nil
}

func cleanUpAbandonedTriggerLastCheckOnRedisNode(connector *DbConnector, client redis.UniversalClient) error {
	lastCheckIterator := client.Scan(connector.context, 0, metricLastCheckKey("*"), 0).Iterator()
	count := 0
	for lastCheckIterator.Next(connector.context) {
		lastCheckKey := lastCheckIterator.Val()
		triggerID := strings.TrimPrefix(lastCheckKey, metricLastCheckKey(""))
		_, err := connector.GetTrigger(triggerID)
		if err == database.ErrNil {
			err = connector.RemoveTriggerLastCheck(triggerID)
			if err != nil {
				return err
			}
			count++
			connector.logger.Debug().
				String("trigger_id", triggerID).
				Msg("Cleaned up last check for trigger")
		} else if err != nil {
			connector.logger.Error().
				String("trigger_id", triggerID).
				Error(err).
				Msg("Error while getting trigger")
		}
	}

	connector.logger.Info().
		Int("count deleted last_check", count).
		Msg("Cleaned up last check for trigger")

	return nil
}

// CleanUpAbandonedTriggerLastCheck cleans up abandoned triggers last check.
func (connector *DbConnector) CleanUpAbandonedTriggerLastCheck() error {
	return connector.callFunc(cleanUpAbandonedTriggerLastCheckOnRedisNode)
}

// SetTriggerCheckMaintenance sets maintenance for whole trigger and to given metrics,
// If CheckData does not contain one of given metrics it will ignore this metric
func (connector *DbConnector) SetTriggerCheckMaintenance(triggerID string, metrics map[string]int64, triggerMaintenance *int64, userLogin string, timeCallMaintenance int64) error {
	ctx := connector.context
	c := *connector.client
	var readingErr error

	lastCheckString, readingErr := c.Get(ctx, metricLastCheckKey(triggerID)).Result()
	if readingErr != nil {
		if readingErr != redis.Nil {
			return readingErr
		}
		return nil
	}

	var lastCheck = moira.CheckData{}
	err := json.Unmarshal([]byte(lastCheckString), &lastCheck)
	if err != nil {
		return fmt.Errorf("failed to parse lastCheck json %s: %s", lastCheckString, err.Error())
	}
	metricsCheck := lastCheck.Metrics
	if len(metricsCheck) > 0 {
		for metric, value := range metrics {
			data, ok := metricsCheck[metric]
			if !ok {
				continue
			}
			moira.SetMaintenanceUserAndTime(&data, value, userLogin, timeCallMaintenance)
			metricsCheck[metric] = data
		}
	}
	if triggerMaintenance != nil {
		moira.SetMaintenanceUserAndTime(&lastCheck, *triggerMaintenance, userLogin, timeCallMaintenance)
	}
	newLastCheck, err := json.Marshal(lastCheck)
	if err != nil {
		return err
	}

	return c.Set(ctx, metricLastCheckKey(triggerID), newLastCheck, redis.KeepTTL).Err()
}

// checkDataScoreChanged returns true if checkData.Score changed since last check
func (connector *DbConnector) checkDataScoreChanged(triggerID string, checkData *moira.CheckData) bool {
	ctx := connector.context
	c := *connector.client

	oldScore, err := c.ZScore(ctx, triggersChecksKey, triggerID).Result()
	if err != nil {
		return true
	}

	return oldScore != float64(checkData.Score)
}

// getTriggersLastCheck returns an array of trigger checks by the passed ids, if the trigger does not exist, it is nil
func (connector *DbConnector) getTriggersLastCheck(triggerIDs []string) ([]*moira.CheckData, error) {
	ctx := connector.context
	pipe := (*connector.client).TxPipeline()

	results := make([]*redis.StringCmd, len(triggerIDs))
	for i, id := range triggerIDs {
		var result *redis.StringCmd
		if id != "" {
			result = pipe.Get(ctx, metricLastCheckKey(id))
		}
		results[i] = result
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	return reply.Checks(results)
}

var badStateTriggersKey = "moira-bad-state-triggers"
var triggersChecksKey = "moira-triggers-checks"

func metricLastCheckKey(triggerID string) string {
	return "moira-metric-last-check:" + triggerID
}
