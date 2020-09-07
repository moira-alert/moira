package redis

import (
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
)

// GetTriggerThrottling get throttling or scheduled notifications delay for given triggerID
func (connector *DbConnector) GetTriggerThrottling(triggerID string) (time.Time, time.Time) {
	c := connector.pool.Get()
	defer c.Close()

	next, _ := redis.Int64(c.Do("GET", notifierNextKey(triggerID)))
	beginning, _ := redis.Int64(c.Do("GET", notifierThrottlingBeginningKey(triggerID)))

	return time.Unix(next, 0), time.Unix(beginning, 0)
}

// SetTriggerThrottling store throttling or scheduled notifications delay for given triggerID
func (connector *DbConnector) SetTriggerThrottling(triggerID string, next time.Time) error {
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("SET", notifierNextKey(triggerID), next.Unix())
	return err
}

// DeleteTriggerThrottling deletes throttling and scheduled notifications delay for given triggerID
func (connector *DbConnector) DeleteTriggerThrottling(triggerID string) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI") //nolint
	c.Send("SET", notifierThrottlingBeginningKey(triggerID), time.Now().Unix()) //nolint
	c.Send("DEL", notifierNextKey(triggerID)) //nolint
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}
	return nil
}

func notifierThrottlingBeginningKey(triggerID string) string {
	return "moira-notifier-throttling-beginning:" + triggerID
}

func notifierNextKey(triggerID string) string {
	return "moira-notifier-next:" + triggerID
}
