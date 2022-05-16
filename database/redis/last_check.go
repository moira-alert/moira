package redis

import (
	"encoding/json"
	"fmt"
	"time"

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
func (connector *DbConnector) SetTriggerLastCheck(triggerID string, checkData *moira.CheckData, isRemote bool) error {
	selfStateCheckCountKey := connector.getSelfStateCheckCountKey(isRemote)
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

func (connector *DbConnector) getSelfStateCheckCountKey(isRemote bool) string {
	if connector.source != Checker {
		return ""
	}
	if isRemote {
		return selfStateRemoteChecksCounterKey
	}
	return selfStateChecksCounterKey
}

// RemoveTriggerLastCheck removes trigger last check data
func (connector *DbConnector) RemoveTriggerLastCheck(triggerID string) error {
	ctx := connector.context
	pipe := (*connector.client).TxPipeline()
	pipe.Del(ctx, metricLastCheckKey(triggerID))
	pipe.ZRem(ctx, triggersChecksKey, triggerID)
	pipe.SRem(ctx, badStateTriggersKey, triggerID)
	pipe.ZAdd(ctx, triggersToReindexKey, &redis.Z{Score: float64(time.Now().Unix()), Member: triggerID})
	_, err := pipe.Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	return nil
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

var badStateTriggersKey = "moira-bad-state-triggers"
var triggersChecksKey = "moira-triggers-checks"

func metricLastCheckKey(triggerID string) string {
	return "moira-metric-last-check:" + triggerID
}
