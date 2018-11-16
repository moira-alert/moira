package index

import (
	"fmt"

	"github.com/blevesearch/bleve"
	"github.com/moira-alert/moira/index/mapping"
)

// SearchTriggers search for triggers in index and returns slice of trigger IDs
func (index *Index) SearchTriggers(filterTags, searchTerms []string, onlyErrors bool) ([]string, error) {
	maxDocuments, _ := index.index.DocCount()
	req := buildSearchRequest(filterTags, searchTerms, onlyErrors, maxDocuments)

	searchResult, err := index.index.Search(req)
	if err != nil {
		return make([]string, 0), err
	}
	if searchResult.Hits.Len() == 0 {
		return make([]string, 0), nil
	}
	triggerIds := make([]string, 0)
	for _, result := range searchResult.Hits {
		triggerIds = append(triggerIds, result.ID)
	}
	return triggerIds, nil
}

func buildSearchRequest(filterTags, searchTerms []string, onlyErrors bool, maxDocuments uint64) *bleve.SearchRequest {
	searchQuery := buildSearchQuery(filterTags, searchTerms, onlyErrors)

	req := bleve.NewSearchRequest(searchQuery)
	req.Size = int(maxDocuments)
	// sorting order:
	// TriggerCheck.Score (desc)
	// Relevance (asc)
	// Trigger.Name (asc)
	req.SortBy([]string{fmt.Sprintf("-%s", mapping.TriggerLastCheckScore.String()), "_score", mapping.TriggerName.String()})

	return req
}
