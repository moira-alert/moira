package redis

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/go-redis/redis/v8"
)

// FetchTriggersToReindex returns trigger IDs updated since 'from' param
// The trigger could be changed by user, or it's score was changed during trigger check
func (connector *DbConnector) FetchTriggersToReindex(from int64) ([]string, error) {
	ctx := connector.context
	c := *connector.client

	rng := &redis.ZRangeBy{Min: strconv.FormatInt(from, 10), Max: "+inf"}
	response, err := c.ZRangeByScore(ctx, triggersToReindexKey, rng).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to fetch triggers to reindex: %s", err.Error())
	}
	if len(response) == 0 {
		return make([]string, 0), nil
	}

	return response, nil
}

// RemoveTriggersToReindex removes outdated triggerIDs from redis
func (connector *DbConnector) RemoveTriggersToReindex(to int64) error {
	ctx := connector.context
	c := *connector.client

	_, err := c.ZRemRangeByScore(ctx, triggersToReindexKey, "-inf", strconv.FormatInt(to, 10)).Result()

	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("failed to remove triggers to reindex: %s", err.Error())
	}

	return nil
}

var triggersToReindexKey = "moira-triggers-to-reindex"
