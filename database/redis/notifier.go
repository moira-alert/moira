package redis

import (
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
)

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
