package reply

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

type searchHighlightStorageElement struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type searchResultStorageElement struct {
	ObjectID   string                          `json:"object_id"`
	Highlights []searchHighlightStorageElement `json:"highlights"`
}

func toSearchResultStorageElement(searchResult moira.SearchResult) searchResultStorageElement {
	result := searchResultStorageElement{
		ObjectID:   searchResult.ObjectID,
		Highlights: make([]searchHighlightStorageElement, 0, len(searchResult.Highlights)),
	}
	for _, highlight := range searchResult.Highlights {
		result.Highlights = append(result.Highlights, searchHighlightStorageElement{
			Field: highlight.Field,
			Value: highlight.Value,
		})
	}
	return result
}

// GetSearchResultBytes is a function that takes a search result converts it to a storage structure and marshal it to JSON.
func GetSearchResultBytes(searchResult moira.SearchResult) ([]byte, error) {
	storageElement := toSearchResultStorageElement(searchResult)
	return json.Marshal(storageElement)
}

func toSearchResult(storageElement searchResultStorageElement) moira.SearchResult {
	result := moira.SearchResult{
		ObjectID:   storageElement.ObjectID,
		Highlights: make([]moira.SearchHighlight, 0, len(storageElement.Highlights)),
	}
	for _, highlight := range storageElement.Highlights {
		result.Highlights = append(result.Highlights, moira.SearchHighlight{
			Field: highlight.Field,
			Value: highlight.Value,
		})
	}
	return result
}

// unmarshalSearchResult is a function that converts redis reply to unmarshalSearchResult.
func unmarshalSearchResult(bytes []byte, err error) (moira.SearchResult, error) {
	var searchResult moira.SearchResult
	storageElement := searchResultStorageElement{}

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return searchResult, database.ErrNil
		}
		return searchResult, fmt.Errorf("failed to read searchResult: %w", err)
	}
	err = json.Unmarshal(bytes, &storageElement)
	if err != nil {
		return searchResult, fmt.Errorf("failed to parse searchResult json %s: %w", string(bytes), err)
	}
	searchResult = toSearchResult(storageElement)
	return searchResult, nil
}

// SearchResults is a function that converts redis reply to slice of SearchResults.
func SearchResults(rep *redis.StringSliceCmd, repTotal *redis.IntCmd) ([]*moira.SearchResult, int64, error) {
	total := repTotal.Val()
	values, err := rep.Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return make([]*moira.SearchResult, 0), 0, nil
		}
		return nil, 0, fmt.Errorf("failed to read SearchResults: %w", err)
	}
	searchResults := make([]*moira.SearchResult, len(values))
	for i, value := range values {
		searchResult, err2 := unmarshalSearchResult([]byte(value), err)
		if err2 != nil && !errors.Is(err2, database.ErrNil) {
			return nil, 0, err2
		}
		if errors.Is(err2, database.ErrNil) {
			searchResults[i] = nil
		} else {
			searchResults[i] = &searchResult
		}
	}
	return searchResults, total, nil
}
