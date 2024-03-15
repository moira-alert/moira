package redis

import (
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis/reply"
)

const triggersSearchResultsExpire = time.Second * 1800

// SaveTriggersSearchResults is a function that takes an ID of pager and saves it to redis.
func (connector *DbConnector) SaveTriggersSearchResults(searchResultsID string, searchResults []*moira.SearchResult) error {
	ctx := connector.context
	pipe := (*connector.client).TxPipeline()

	resultsID := triggersSearchResultsKey(searchResultsID)
	for _, searchResult := range searchResults {
		var marshalled []byte
		marshalled, err := reply.GetSearchResultBytes(*searchResult)
		if err != nil {
			return fmt.Errorf("marshall error: %w", err)
		}
		if err = pipe.RPush(ctx, resultsID, marshalled).Err(); err != nil {
			return fmt.Errorf("failed to PUSH: %w", err)
		}
	}
	if err := pipe.Expire(ctx, resultsID, triggersSearchResultsExpire).Err(); err != nil {
		return fmt.Errorf("failed to set expire time: %w", err)
	}
	response, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to EXEC: %w", err)
	}
	connector.logger.Debug().
		Interface("response", response).
		Msg("EXEC response")
	return nil
}

// GetTriggersSearchResults is a function that receives a saved pager from redis.
func (connector *DbConnector) GetTriggersSearchResults(searchResultsID string, page, size int64) ([]*moira.SearchResult, int64, error) {
	ctx := connector.context
	pipe := (*connector.client).TxPipeline()

	var from, to int64 = 0, -1
	if size > 0 {
		from = page * size
		to = from + size - 1
	}

	resultsID := triggersSearchResultsKey(searchResultsID)

	pipe.LRange(ctx, resultsID, from, to)
	pipe.LLen(ctx, resultsID)
	response, err := pipe.Exec(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to EXEC: %w", err)
	}

	rangeResult := response[0].(*redis.StringSliceCmd)
	lenResult := response[1].(*redis.IntCmd)
	if len(rangeResult.Val()) == 0 {
		return make([]*moira.SearchResult, 0), lenResult.Val(), nil
	}
	return reply.SearchResults(rangeResult, lenResult)
}

// IsTriggersSearchResultsExist is a function that checks if there exists pager for triggers search by it's ID.
func (connector *DbConnector) IsTriggersSearchResultsExist(pagerID string) (bool, error) {
	ctx := connector.context
	c := *connector.client

	pagerIDKey := triggersSearchResultsKey(pagerID)
	response, err := c.Exists(ctx, pagerIDKey).Result()

	if err != nil {
		return false, fmt.Errorf("failed to check if pager exists: %w", err)
	}

	return response == 1, nil
}

// DeleteTriggersSearchResults is a function that checks if there exists pager for triggers search by it's ID.
func (connector *DbConnector) DeleteTriggersSearchResults(pagerID string) error {
	ctx := connector.context
	c := *connector.client

	pagerIDKey := triggersSearchResultsKey(pagerID)
	err := c.Del(ctx, pagerIDKey).Err()

	if err != nil {
		return fmt.Errorf("failed to check if pager exists: %w", err)
	}
	return nil
}

func triggersSearchResultsKey(searchResultsID string) string {
	return fmt.Sprintf("moira-triggersSearchResults:%s", searchResultsID)
}
