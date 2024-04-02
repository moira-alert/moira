package bleve

import (
	"fmt"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/mapping"
)

// Search gets search params and returns triggerIDs in following order:
// TriggerCheck.Score (desc).
// Relevance (asc).
// Trigger.Name (asc).
func (index *TriggerIndex) Search(options moira.SearchOptions) (searchResults []*moira.SearchResult, total int64, err error) {
	if options.Size < 0 {
		options.Page = 0
		docs, _ := index.index.DocCount()
		options.Size = int64(docs)
	}

	req := buildSearchRequest(options)

	searchResult, err := index.index.Search(req)
	if err != nil {
		return
	}
	total = int64(searchResult.Total)
	if searchResult.Hits.Len() == 0 {
		return
	}

	for _, result := range searchResult.Hits {
		highlights := getHighlights(result.Fragments, mapping.TriggerName, mapping.TriggerDesc)
		triggerSearchResult := moira.SearchResult{
			ObjectID:   result.ID,
			Highlights: highlights,
		}
		searchResults = append(searchResults, &triggerSearchResult)
	}
	return
}

func getHighlights(fragmentsMap search.FieldFragmentMap, triggerFields ...mapping.FieldData) []moira.SearchHighlight {
	highlights := make([]moira.SearchHighlight, 0)
	for _, triggerField := range triggerFields {
		var highlightValue string
		if fragments, ok := fragmentsMap[triggerField.GetName()]; ok {
			for _, fragment := range fragments {
				highlightValue += fragment
			}
			highlights = append(highlights, moira.SearchHighlight{
				Field: triggerField.GetTagValue(),
				Value: highlightValue,
			})
		}
	}
	return highlights
}

func buildSearchRequest(options moira.SearchOptions) *bleve.SearchRequest {
	searchQuery := buildSearchQuery(options)

	from := options.Page * options.Size
	req := bleve.NewSearchRequestOptions(searchQuery, int(options.Size), int(from), false)

	if options.NeedSortingOnlyById {
		req.SortBy([]string{mapping.TriggerID.GetName()})
	} else {
		// sorting order:
		req.SortBy([]string{
			// TriggerCheck.Score (desc)
			fmt.Sprintf("-%s", mapping.TriggerLastCheckScore.GetName()),
			// Relevance (asc)
			"-_score",
			// Trigger.Name (asc)
			mapping.TriggerName.GetName(),
		})
	}

	req.Highlight = bleve.NewHighlight()

	return req
}
