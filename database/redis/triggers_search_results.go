package redis

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis/reply"
)

const triggersSearchResultsExpireSeconds = 1800

// SaveTriggersSearchResults is a function that takes an ID of pager and saves it to redis
func (connector *DbConnector) SaveTriggersSearchResults(searchResultsID string, searchResults []*moira.SearchResult) error {
	c := connector.pool.Get()
	defer c.Close()

	resultsID := triggersSearchResultsKey(searchResultsID)
	if err := c.Send("MULTI"); err != nil {
		return fmt.Errorf("failed to MULTI: %w", err)
	}
	for _, searchResult := range searchResults {
		var marshalled []byte
		marshalled, err := reply.GetSearchResultBytes(*searchResult)
		if err != nil {
			return fmt.Errorf("marshall error: %w", err)
		}
		if err := c.Send("RPUSH", resultsID, marshalled); err != nil {
			return fmt.Errorf("failed to PUSH: %w", err)
		}
	}
	if err := c.Send("EXPIRE", resultsID, triggersSearchResultsExpireSeconds); err != nil {
		return fmt.Errorf("failed to set expire time: %w", err)
	}
	response, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("failed to EXEC: %w", err)
	}
	connector.logger.Debugf("EXEC response: %v", response)
	return nil
}

// GetTriggersSearchResults is a function that receives a saved pager from redis
func (connector *DbConnector) GetTriggersSearchResults(searchResultsID string, page, size int64) ([]*moira.SearchResult, int64, error) {
	c := connector.pool.Get()
	defer c.Close()

	var from, to int64 = 0, -1
	if size > 0 {
		from = page * size
		to = from + size - 1
	}

	resultsID := triggersSearchResultsKey(searchResultsID)

	c.Send("MULTI")                       //nolint
	c.Send("LRANGE", resultsID, from, to) //nolint
	c.Send("LLEN", resultsID)             //nolint
	response, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to EXEC: %w", err)
	}
	if len(response) == 0 {
		return make([]*moira.SearchResult, 0), 0, nil
	}
	return reply.SearchResults(response[0], response[1], nil)
}

// IsTriggersSearchResultsExist is a function that checks if there exists pager for triggers search by it's ID
func (connector *DbConnector) IsTriggersSearchResultsExist(pagerID string) (bool, error) {
	c := connector.pool.Get()
	defer c.Close()

	pagerIDKey := triggersSearchResultsKey(pagerID)
	reply, err := c.Do("EXISTS", pagerIDKey)

	result, err := redis.Bool(reply, err)
	if err != nil {
		return false, fmt.Errorf("failed to check if pager exists: %w", err)
	}
	return result, nil
}

// DeleteTriggersSearchResults is a function that checks if there exists pager for triggers search by it's ID
func (connector *DbConnector) DeleteTriggersSearchResults(pagerID string) error {
	c := connector.pool.Get()
	defer c.Close()

	pagerIDKey := triggersSearchResultsKey(pagerID)
	_, err := c.Do("DEL", pagerIDKey)

	if err != nil {
		return fmt.Errorf("failed to check if pager exists: %w", err)
	}
	return nil
}

func triggersSearchResultsKey(searchResultsID string) string {
	return fmt.Sprintf("moira-triggersSearchResults:%s", searchResultsID)
}
