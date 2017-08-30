package redis

import (
	"encoding/json"
	"fmt"

	"github.com/garyburd/redigo/redis"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database/redis/reply"
)

//GetTriggerLastCheck gets trigger last check data by given triggerID, if no value, return database.ErrNil error
func (connector *DbConnector) GetTriggerLastCheck(triggerID string) (moira.CheckData, error) {
	c := connector.pool.Get()
	defer c.Close()
	return reply.Check(c.Do("GET", moiraMetricLastCheck(triggerID)))
}

//SetTriggerLastCheck sets trigger last check data
func (connector *DbConnector) SetTriggerLastCheck(triggerID string, checkData *moira.CheckData) error {
	bytes, err := json.Marshal(checkData)
	if err != nil {
		return err
	}
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	c.Send("SET", moiraMetricLastCheck(triggerID), bytes)
	c.Send("ZADD", moiraTriggersChecks, checkData.Score, triggerID)
	c.Send("INCR", moiraSelfStateChecksCounter)
	if checkData.Score > 0 {
		c.Send("SADD", moiraBadStateTriggers, triggerID)
	} else {
		c.Send("SREM", moiraBadStateTriggers, triggerID)
	}
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

//GetTriggerCheckIDs gets checked triggerIDs, sorted from max to min check score
func (connector *DbConnector) GetTriggerCheckIDs() ([]string, int64, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("ZREVRANGE", moiraTriggersChecks, 0, -1)
	c.Send("ZCARD", moiraTriggersChecks)
	rawResponse, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, 0, err
	}
	triggerIds, err := redis.Strings(rawResponse[0], nil)
	if err != nil {
		return nil, 0, err
	}
	total, err := redis.Int64(rawResponse[1], nil)
	if err != nil {
		return nil, 0, err
	}
	return triggerIds, total, nil
}

//SetTriggerCheckMetricsMaintenance sets to given metrics throttling timestamps, if during the update lastCheck was updated, try update again
func (connector *DbConnector) SetTriggerCheckMetricsMaintenance(triggerID string, metrics map[string]int64) error {
	c := connector.pool.Get()
	defer c.Close()
	var readingErr error

	lastCheckString, readingErr := redis.String(c.Do("GET", moiraMetricLastCheck(triggerID)))
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
					data = moira.MetricState{}
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
		prev, readingErr = redis.String(c.Do("GETSET", moiraMetricLastCheck(triggerID), newLastCheck))
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

var moiraBadStateTriggers = "moira-bad-state-triggers"
var moiraTriggersChecks = "moira-triggers-checks"

func moiraMetricLastCheck(triggerID string) string {
	return fmt.Sprintf("moira-metric-last-check:%s", triggerID)
}
