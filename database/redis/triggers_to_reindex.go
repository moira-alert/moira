package redis

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
)

// FetchTriggersToReindex returns trigger IDs updated since 'from' param
// The trigger could be changed by user, or it's score was changed during trigger check
func (connector *DbConnector) FetchTriggersToReindex(from int64) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	response, err := redis.Strings(c.Do("ZRANGEBYSCORE", triggersToReindexKey, from, "+inf"))

	if err != nil {
		return nil, fmt.Errorf("failed to fetch triggers to reindex: %w", err)
	}
	if len(response) == 0 {
		return make([]string, 0), nil
	}

	return response, nil
}

// RemoveTriggersToReindex removes outdated triggerIDs from redis
func (connector *DbConnector) RemoveTriggersToReindex(to int64) error {
	c := connector.pool.Get()
	defer c.Close()

	err := c.Send("ZREMRANGEBYSCORE", triggersToReindexKey, "-inf", to)
	if err != nil {
		if err == redis.ErrNil {
			return nil
		}
		return fmt.Errorf("failed to remove triggers to reindex: %w", err)
	}
	return nil
}

var triggersToReindexKey = "moira-triggers-to-reindex"
