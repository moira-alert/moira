package redis

import (
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// GetTriggerThrottling get throttling or scheduled notifications delay for given triggerID
func (connector *DbConnector) GetTriggerThrottling(triggerID string) (time.Time, time.Time) {
	c := *connector.client

	next, _ := c.Get(connector.context, notifierNextKey(triggerID)).Int64()
	beginning, _ := c.Get(connector.context, notifierThrottlingBeginningKey(triggerID)).Int64()

	return time.Unix(next, 0), time.Unix(beginning, 0)
}

// SetTriggerThrottling store throttling or scheduled notifications delay for given triggerID
func (connector *DbConnector) SetTriggerThrottling(triggerID string, next time.Time) error {
	c := *connector.client
	err := c.Set(connector.context, notifierNextKey(triggerID), next.Unix(), redis.KeepTTL).Err()
	return err
}

// DeleteTriggerThrottling deletes throttling and scheduled notifications delay for given triggerID
func (connector *DbConnector) DeleteTriggerThrottling(triggerID string) error {
	c := *connector.client

	pipe := c.TxPipeline()
	pipe.Set(connector.context, notifierThrottlingBeginningKey(triggerID), time.Now().Unix(), redis.KeepTTL)
	pipe.Del(connector.context, notifierNextKey(triggerID))
	_, err := pipe.Exec(connector.context)
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
