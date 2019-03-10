package bleve

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/blevesearch/bleve"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/mapping"
)

// Search gets search params and returns triggerIDs in following order:
// TriggerCheck.Score (desc)
// Relevance (asc)
// Trigger.Name (asc)
func (index *TriggerIndex) Search(filterTags []string, searchString string, onlyErrors bool, page int64, size int64) (searchResults []*moira.SearchResult, total int64, err error) {
	if size < 0 {
		page = 0
		docs, _ := index.index.DocCount()
		size = int64(docs)
	}

	req := buildSearchRequest(filterTags, searchString, onlyErrors, int(page), int(size))

	searchResult, err := index.index.Search(req)
	if err != nil {
		return
	}
	total = int64(searchResult.Total)
	if searchResult.Hits.Len() == 0 {
		return
	}
	for _, result := range searchResult.Hits {
		highLights := make([]moira.SearchHighLight, 0)
		if nameFragments, ok := result.Fragments[mapping.TriggerName.String()]; ok {
			var nameHighLight string
			for _, fragment := range nameFragments {
				nameHighLight += fragment
			}
			highLights = append(highLights, moira.SearchHighLight{
				Field: mapping.TriggerName.String(),
				Value: nameHighLight,
			})
		}
		if descFragments, ok := result.Fragments[mapping.TriggerDesc.String()]; ok {
			var descHighLight string
			for _, fragment := range descFragments {
				descHighLight += fragment
			}
			highLights = append(highLights, moira.SearchHighLight{
				Field: mapping.TriggerDesc.String(),
				Value: descHighLight,
			})
		}
		triggerSearchResult := moira.SearchResult{
			ObjectID:   result.ID,
			HighLights: highLights,
		}
		searchResults = append(searchResults, &triggerSearchResult)
	}
	return
}

func buildSearchRequest(filterTags []string, searchString string, onlyErrors bool, page, size int) *bleve.SearchRequest {

	searchTerms := splitStringToTerms(searchString)
	searchQuery := buildSearchQuery(filterTags, searchTerms, onlyErrors)

	from := page * size
	req := bleve.NewSearchRequestOptions(searchQuery, size, from, false)
	// sorting order:
	// TriggerCheck.Score (desc)
	// Relevance (asc)
	// Trigger.Name (asc)
	req.SortBy([]string{fmt.Sprintf("-%s", mapping.TriggerLastCheckScore.String()), "_score", mapping.TriggerName.String()})
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
