package redis

import (
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
)

// AcquireTriggerCheckLock sets trigger lock by given id. If lock does not take, try again and repeat it for given attempts
func (connector *DbConnector) AcquireTriggerCheckLock(triggerID string, timeout int) error {
	acquired, err := connector.SetTriggerCheckLock(triggerID)
	if err != nil {
		return err
	}
	count := 0
	for !acquired && count < timeout {
		count++
		<-time.After(time.Millisecond * 500)
		acquired, err = connector.SetTriggerCheckLock(triggerID)
		if err != nil {
			return err
		}
	}
	if !acquired {
		return fmt.Errorf("Can not acquire trigger lock in %v seconds", timeout)
	}
	return nil
}

// SetTriggerCheckLock create to database lock object with 30sec TTL and return true if object successfully created, or false if object already exists
func (connector *DbConnector) SetTriggerCheckLock(triggerID string) (bool, error) {
	c := connector.pool.Get()
	defer c.Close()
	_, err := redis.String(c.Do("SET", fmt.Sprintf("moira-metric-check-lock:%s", triggerID), time.Now().Unix(), "EX", 30, "NX"))
	if err != nil {
		if err == redis.ErrNil {
			return false, nil
		}
		return false, fmt.Errorf("Failed to set metric-check-lock:%s : %s", triggerID, err.Error())
	}
	return true, nil
}

// DeleteTriggerCheckLock deletes trigger check lock for given triggerID
func (connector *DbConnector) DeleteTriggerCheckLock(triggerID string) error {
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("DEL", fmt.Sprintf("moira-metric-check-lock:%s", triggerID))
	if err != nil {
		return fmt.Errorf("Failed to delete trigger check lock:%s : %s", triggerID, err.Error())
	}
	return nil
}
