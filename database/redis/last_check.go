package redis

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis/reply"
)

// GetTriggerLastCheck gets trigger last check data by given triggerID, if no value, return database.ErrNil error
func (connector *DbConnector) GetTriggerLastCheck(triggerID string) (moira.CheckData, error) {
	c := connector.pool.Get()
	defer c.Close()
	return reply.Check(c.Do("GET", metricLastCheckKey(triggerID)))
}

// SetTriggerLastCheck sets trigger last check data
func (connector *DbConnector) SetTriggerLastCheck(triggerID string, checkData *moira.CheckData, isRemote bool) error {
	if isRemote {
		return connector.setTriggerLastCheckAndUpdateProperCounter(triggerID, checkData, selfStateRemoteChecksCounterKey)
	}
	return connector.setTriggerLastCheckAndUpdateProperCounter(triggerID, checkData, selfStateChecksCounterKey)
}

func (connector *DbConnector) setTriggerLastCheckAndUpdateProperCounter(triggerID string, checkData *moira.CheckData, selfStateCheckCountKey string) error {
	bytes, err := json.Marshal(checkData)
	if err != nil {
		return err
	}

	triggerNeedToReindex := connector.checkDataScoreChanged(triggerID, checkData)

	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	c.Send("SET", metricLastCheckKey(triggerID), bytes)
	c.Send("INCR", selfStateCheckCountKey)
	if triggerNeedToReindex {
		c.Send("ZADD", triggersToReindexKey, time.Now().Unix(), triggerID)
	}
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

// RemoveTriggerLastCheck removes trigger last check data
func (connector *DbConnector) RemoveTriggerLastCheck(triggerID string) error {
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	c.Send("DEL", metricLastCheckKey(triggerID))
	c.Send("ZADD", triggersToReindexKey, time.Now().Unix(), triggerID)
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}

	return nil
}

// SetTriggerCheckMetricsMaintenance sets to given metrics throttling timestamps,
// If during the update lastCheck was updated from another place, try update again
// If CheckData does not contain one of given metrics it will ignore this metric
func (connector *DbConnector) SetTriggerCheckMetricsMaintenance(triggerID string, metrics map[string]int64) error {
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
			return fmt.Errorf("Failed to parse lastCheck json %s: %s", lastCheckString, err.Error())
		}
		metricsCheck := lastCheck.Metrics
		if len(metricsCheck) > 0 {
			for metric, value := range metrics {
				data, ok := metricsCheck[metric]
				if !ok {
					continue
				}
				data.Maintenance = value
				metricsCheck[metric] = data
			}
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

	oldLastCheck, err := reply.Check(c.Do("GET", metricLastCheckKey(triggerID)))
	if err != nil {
		return true
	}
	oldScore := oldLastCheck.Score

	return oldScore != checkData.Score
}

func metricLastCheckKey(triggerID string) string {
	return fmt.Sprintf("moira-metric-last-check:%s", triggerID)
}
