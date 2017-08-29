package redis

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
)

// GetNotificationTrigger returns trigger data
func (connector *DbConnector) GetNotificationTrigger(id string) (moira.TriggerData, error) {
	c := connector.pool.Get()
	defer c.Close()

	var trigger moira.TriggerData

	triggerString, err := redis.Bytes(c.Do("GET", fmt.Sprintf("moira-trigger:%s", id)))
	if err != nil {
		return trigger, fmt.Errorf("Failed to get trigger data for id %s: %s", id, err.Error())
	}
	if err := json.Unmarshal(triggerString, &trigger); err != nil {
		return trigger, fmt.Errorf("Failed to parse trigger json %s: %s", triggerString, err.Error())
	}

	return trigger, nil
}

// GetTriggerTags returns trigger tags
func (connector *DbConnector) GetTriggerTags(triggerID string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	var tags []string

	values, err := redis.Values(c.Do("SMEMBERS", fmt.Sprintf("moira-trigger-tags:%s", triggerID)))
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve tags for trigger id %s: %s", triggerID, err.Error())
	}
	if err := redis.ScanSlice(values, &tags); err != nil {
		return nil, fmt.Errorf("Failed to retrieve tags for trigger id %s: %s", triggerID, err.Error())
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("No tags found for trigger id %s", triggerID)
	}
	return tags, nil
}

// GetTriggerThrottlingTimestamps get throttling or scheduled notifications delay for given triggerID
func (connector *DbConnector) GetTriggerThrottlingTimestamps(triggerID string) (time.Time, time.Time) {
	c := connector.pool.Get()
	defer c.Close()

	next, _ := redis.Int64(c.Do("GET", fmt.Sprintf("moira-notifier-next:%s", triggerID)))
	beginning, _ := redis.Int64(c.Do("GET", fmt.Sprintf("moira-notifier-throttling-beginning:%s", triggerID)))

	return time.Unix(next, 0), time.Unix(beginning, 0)
}

// SetTriggerThrottlingTimestamp store throttling or scheduled notifications delay for given triggerID
func (connector *DbConnector) SetTriggerThrottlingTimestamp(triggerID string, next time.Time) error {
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("SET", fmt.Sprintf("moira-notifier-next:%s", triggerID), next.Unix())
	return err
}

// GetMetricsCount - return metrics count received by Moira-Cache
func (connector *DbConnector) GetMetricsCount() (int64, error) {
	c := connector.pool.Get()
	defer c.Close()
	ts, err := redis.Int64(c.Do("GET", "moira-selfstate:metrics-heartbeat"))
	if err == redis.ErrNil {
		return 0, nil
	}
	return ts, err
}

// GetChecksCount - return checks count by Moira-Checker
func (connector *DbConnector) GetChecksCount() (int64, error) {
	c := connector.pool.Get()
	defer c.Close()
	ts, err := redis.Int64(c.Do("GET", "moira-selfstate:checks-counter"))
	if err == redis.ErrNil {
		return 0, nil
	}
	return ts, err
}
