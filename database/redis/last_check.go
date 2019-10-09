package redis

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis/reply"
)

// GetTriggerLastCheck gets trigger last check data by given triggerID, if no value, return database.ErrNil error
func (connector *DbConnector) GetTriggerLastCheck(triggerID string) (moira.CheckData, error) {
	c := connector.pool.Get()
	defer c.Close()
	lastCheck, err := reply.Check(c.Do("GET", metricLastCheckKey(triggerID)))
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

	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	c.Send("SET", metricLastCheckKey(triggerID), bytes)
	c.Send("ZADD", triggersChecksKey, checkData.Score, triggerID)
	if selfStateCheckCountKey != "" {
		c.Send("INCR", selfStateCheckCountKey)
	}
	if checkData.Score > 0 {
		c.Send("SADD", badStateTriggersKey, triggerID)
	} else {
		c.Send("SREM", badStateTriggersKey, triggerID)
	}
	if triggerNeedToReindex {
		c.Send("ZADD", triggersToReindexKey, time.Now().Unix(), triggerID)
	}
	_, err = c.Do("EXEC")
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
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	c.Send("DEL", metricLastCheckKey(triggerID))
	c.Send("ZREM", triggersChecksKey, triggerID)
	c.Send("SREM", badStateTriggersKey, triggerID)
	c.Send("ZADD", triggersToReindexKey, time.Now().Unix(), triggerID)
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	return nil
}

// SetTriggerCheckMaintenance sets maintenance for whole trigger and to given metrics,
// If during the update lastCheck was updated from another place, try update again
// If CheckData does not contain one of given metrics it will ignore this metric
func (connector *DbConnector) SetTriggerCheckMaintenance(triggerID string, metrics map[string]int64, triggerMaintenance *int64, userLogin string, timeCallMaintenance int64) error {
	c := connector.pool.Get()
	defer c.Close()
	var readingErr error

	lastCheckString, readingErr := redis.String(c.Do("GET", metricLastCheckKey(triggerID)))
	if readingErr != nil && readingErr != redis.ErrNil {
		return readingErr
	}
	for readingErr != redis.ErrNil {
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

		var prev string
		prev, readingErr = redis.String(c.Do("GETSET", metricLastCheckKey(triggerID), newLastCheck))
		if readingErr != nil && readingErr != redis.ErrNil {
			return readingErr
		}
		if prev == lastCheckString {
			break
		}
		lastCheckString = prev
	}
	return nil
}

// checkDataScoreChanged returns true if checkData.Score changed since last check
func (connector *DbConnector) checkDataScoreChanged(triggerID string, checkData *moira.CheckData) bool {
	c := connector.pool.Get()
	defer c.Close()

	oldScore, err := redis.Int64(c.Do("ZSCORE", triggersChecksKey, triggerID))
	if err != nil {
		return true
	}

	return oldScore != checkData.Score
}

var badStateTriggersKey = "moira-bad-state-triggers"
var triggersChecksKey = "moira-triggers-checks"

func metricLastCheckKey(triggerID string) string {
	return "moira-metric-last-check:" + triggerID
}
