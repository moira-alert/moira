package redis

import (
	"fmt"

	"github.com/garyburd/redigo/redis"
)

// FetchTriggersToReindex returns []triggerID of triggers needed to update. It is used for full-text search index
// It returns triggerIDs from 'from' param to a current time
func (connector *DbConnector) FetchTriggersToReindex(from int64) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	response, err := redis.Strings(c.Do("ZRANGEBYSCORE", triggersToReindexKey, from, "+inf"))

	if err != nil {
		return nil, fmt.Errorf("failed to fetch triggers to reindex: %s", err.Error())
	}
	if len(response) == 0 {
		return make([]string, 0), nil
	}

	return response, nil
}

// RemoveTriggersToReindex removes outdated triggerIDs from redis. It is used for full-text search index
// It removes triggerIDs from the beginning of time to 'to' param
func (connector *DbConnector) RemoveTriggersToReindex(to int64) error {
	c := connector.pool.Get()
	defer c.Close()

	err := c.Send("ZREMRANGEBYSCORE", triggersToReindexKey, "-inf", to)
	if err != nil {
		if err == redis.ErrNil {
			return nil
		}
		return fmt.Errorf("failed to remove triggers to reindex: %s", err.Error())
	}
	return nil
}

var triggersToReindexKey = "moira-triggers-to-reindex"
