package redis

import (
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira/database"
)

// AddTriggersToUpdate adds triggerID to redis. It is used for full-text search index
func (connector *DbConnector) AddTriggersToUpdate(triggerIDs ...string) error {
	if len(triggerIDs) == 0 {
		return nil
	}

	c := connector.pool.Get()
	defer c.Close()

	unixNow := time.Now().Unix()

	c.Send("MULTI")
	for _, triggerID := range triggerIDs {
		c.Send("ZADD", triggersToUpdateKey, unixNow, triggerID)
	}

	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("failed to add triggers to update: %s", err.Error())
	}
	return nil
}

// FetchTriggersToUpdate returns []triggerID of triggers needed to update. It is used for full-text search index
// It returns triggerIDs from 'from' param to a current time
func (connector *DbConnector) FetchTriggersToUpdate(from int64) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	response, err := redis.Strings(c.Do("ZRANGEBYSCORE", triggersToUpdateKey, from, "+inf"))

	if err != nil {
		return nil, fmt.Errorf("failed to fetch triggers to update: %s", err)
	}
	if len(response) == 0 {
		return make([]string, 0), nil
	}

	return response, nil
}

// RemoveTriggersToUpdate removes outdated triggerIDs from redis. It is used for full-text search index
// It removes triggerIDs from the beginning of time to 'to' param
func (connector *DbConnector) RemoveTriggersToUpdate(to int64) error {
	c := connector.pool.Get()
	defer c.Close()

	err := c.Send("ZREMRANGEBYSCORE", triggersToUpdateKey, "-inf", to)
	if err == redis.ErrNil {
		err = database.ErrNil
	}
	return err
}

var triggersToUpdateKey = "moira-triggers-to-update"
