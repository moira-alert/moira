package bleve

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"

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

	fieldNames := []string{mapping.TriggerName.String(), mapping.TriggerDesc.String()}
	fieldTags := getTagsByFieldNames(fieldNames)

	for _, result := range searchResult.Hits {
		highlights := getHighlights(result.Fragments, fieldNames, fieldTags)
		triggerSearchResult := moira.SearchResult{
			ObjectID:   result.ID,
			Highlights: highlights,
		}
		searchResults = append(searchResults, &triggerSearchResult)
	}
	return
}

func getHighlights(fragmentsMap search.FieldFragmentMap, fieldNames, fieldTags []string) []moira.SearchHighlight {
	highlights := make([]moira.SearchHighlight, 0)
	for fieldInd := range fieldNames {
		var highlightValue string
		fieldName, fieldTag := fieldNames[fieldInd], fieldTags[fieldInd]
		if fragments, ok := fragmentsMap[fieldName]; ok {
			for _, fragment := range fragments {
				highlightValue += fragment
			}
			highlights = append(highlights, moira.SearchHighlight{
				Field: fieldTag,
				Value: highlightValue,
			})
		}
	}
	return highlights
}

// getTagsByFieldNames returns collections of corresponding JSON tags for given trigger fields
// if there is no tag found field name will be added. Output is same length as input.
func getTagsByFieldNames(fieldNames []string) []string {
	var trigger moira.Trigger
	fieldTags := make([]string, 0, len(fieldNames))
	for _, fieldName := range fieldNames {
		var fieldTag string
		if field, ok := reflect.TypeOf(&trigger).Elem().FieldByName(fieldName); ok {
			fieldTag = field.Tag.Get("json")
			fieldTag = strings.Replace(fieldTag, ",omitempty", "", -1)
		} else {
			fieldTag = fieldName
		}
		fieldTags = append(fieldTags, fieldTag)
	}
	return fieldTags
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
