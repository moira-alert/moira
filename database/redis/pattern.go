package redis

import (
	"context"
	"strings"

	"github.com/go-redis/redis/v8"
)

// UpdatePatternList temporary func
func (connector *DbConnector) UpdatePatternList() error {
	client := *connector.client

	switch c := client.(type) {
	case *redis.ClusterClient:
		err := c.ForEachMaster(connector.context, func(ctx context.Context, shard *redis.Client) error {
			err := miniPatternListUpdate(connector, shard)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	default:
		err := miniPatternListUpdate(connector, c)
		if err != nil {
			return err
		}
	}

	return nil
}

func miniPatternListUpdate(connector *DbConnector, client redis.UniversalClient) error {
	allPatterns, err := connector.GetPatterns()
	if err != nil {
		return err
	}

	connector.logger.Clone().
		Int("count", len(allPatterns)).
		Info("patterns before update")

	patternMap := make(map[string]bool)
	for _, pattern := range allPatterns {
		patternMap[pattern] = true
	}

	totalWrittenPatterns := 0

	patternTriggerIterator := client.Scan(connector.context, 0, patternTriggersKey("*"), 0).Iterator()
	for patternTriggerIterator.Next(connector.context) { // nolint
		pattern := strings.TrimPrefix(patternTriggerIterator.Val(), patternTriggersKey(""))
		patternMetrics, err := connector.GetPatternMetrics(pattern)
		if err != nil {
			connector.logger.Clone().
				String("pattern", pattern).
				Errorf("can not get pattern metrics, got err: %v", err)
			continue
		}

		pipe := (*connector.client).TxPipeline()
		if len(patternMetrics) == 0 && !patternMap[pattern] {
			connector.logger.Clone().
				String("pattern", pattern).
				Info("added pattern into list")
			totalWrittenPatterns++
			pipe.SAdd(connector.context, patternsListKey, pattern)
		}

		_, err = pipe.Exec(connector.context)
		if err != nil {
			connector.logger.Clone().
				Errorf("can not exec pip for %v, got err: %v", patternMetrics, err)
			continue
		}
	}

	connector.logger.Clone().
		Int("count", totalWrittenPatterns).
		Info("total written patterns")

	allPatterns, err = connector.GetPatterns()
	if err != nil {
		return err
	}

	connector.logger.Clone().
		Int("count", len(allPatterns)).
		Info("patterns after update")

	return nil
}
