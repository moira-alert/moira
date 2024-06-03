package redis

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// AcquireTriggerCheckLock sets trigger lock by given id. If lock does not take, try again and repeat it for given attempts.
func (connector *DbConnector) AcquireTriggerCheckLock(triggerID string, maxAttemptsCount int) error {
	acquired, err := connector.SetTriggerCheckLock(triggerID)
	if err != nil {
		return err
	}
	attemptsCount := 0
	for !acquired && attemptsCount < maxAttemptsCount {
		attemptsCount++
		<-time.After(time.Second) //nolint
		acquired, err = connector.SetTriggerCheckLock(triggerID)
		if err != nil {
			return err
		}
	}
	if !acquired {
		return fmt.Errorf("can not acquire trigger lock in %v attempts", maxAttemptsCount)
	}
	return nil
}

// SetTriggerCheckLock create to database lock object with 30sec TTL and return true if object successfully created, or false if object already exists.
func (connector *DbConnector) SetTriggerCheckLock(triggerID string) (bool, error) {
	c := *connector.client
	err := c.SetArgs(connector.context, metricCheckLockKey(triggerID), time.Now().Unix(), redis.SetArgs{TTL: 30 * time.Second, Mode: "NX"}).Err()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, fmt.Errorf("failed to set check lock: %s error: %s", triggerID, err.Error())
	}
	return true, nil
}

// DeleteTriggerCheckLock deletes trigger check lock for given triggerID.
func (connector *DbConnector) DeleteTriggerCheckLock(triggerID string) error {
	c := *connector.client
	err := c.Del(connector.context, metricCheckLockKey(triggerID)).Err()
	if err != nil {
		return fmt.Errorf("failed to delete trigger check lock: %s error: %s", triggerID, err.Error())
	}
	return nil
}

// ReleaseTriggerCheckLock deletes trigger check lock for given triggerID and logs an error if needed.
func (connector *DbConnector) ReleaseTriggerCheckLock(triggerID string) {
	if err := connector.DeleteTriggerCheckLock(triggerID); err != nil {
		connector.logger.Warning().
			Error(err).
			Msg("Error on releasing trigger check lock")
	}
}

func metricCheckLockKey(triggerID string) string {
	return "moira-metric-check-lock:" + triggerID
}
