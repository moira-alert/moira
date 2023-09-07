package bleve

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/mapping"
)

// Search gets search params and returns triggerIDs in following order:
// TriggerCheck.Score (desc)
// Relevance (asc)
// Trigger.Name (asc)
func (index *TriggerIndex) Search(filterTags []string, searchString string, onlyErrors bool, page int64, size int64, createdBy string) (searchResults []*moira.SearchResult, total int64, err error) {
	if size < 0 {
		page = 0
		docs, _ := index.index.DocCount()
		size = int64(docs)
	}

	req := buildSearchRequest(filterTags, searchString, onlyErrors, int(page), int(size), createdBy)

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

func buildSearchRequest(filterTags []string, searchString string, onlyErrors bool, page, size int, createdBy string) *bleve.SearchRequest {
	searchTerms := splitStringToTerms(searchString)
	searchQuery := buildSearchQuery(filterTags, searchTerms, onlyErrors, createdBy)

	from := page * size
	req := bleve.NewSearchRequestOptions(searchQuery, size, from, false)
	// sorting order:
	// TriggerCheck.Score (desc)
	// Relevance (asc)
	// Trigger.Name (asc)
	req.SortBy([]string{fmt.Sprintf("-%s", mapping.TriggerLastCheckScore.GetName()), "-_score", mapping.TriggerName.GetName()})
	req.Highlight = bleve.NewHighlight()

	return req
}

func splitStringToTerms(searchString string) (searchTerms []string) {
	searchString = escapeString(searchString)

	return strings.Fields(searchString)
}

func escapeString(original string) (escaped string) {
	return regexp.MustCompile(`[|+\-=&<>!(){}\[\]^"'~*?\\/.,:;_@]`).ReplaceAllString(original, " ")
}
