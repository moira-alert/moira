package bluge

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/search"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/mapping"
)

func (index *TriggerIndex) Search(filterTags []string, searchString string, onlyErrors bool, page int64, size int64) (searchResults []*moira.SearchResult, total int64, err error) {
	reader, err := index.writer.Reader()
	if err != nil {
		return
	}

	if size < 0 {
		page = 0
		var count uint64
		count, err = reader.Count()
		if err != nil {
			return
		}
		size = int64(count)
	}

	req := buildSearchRequest(filterTags, searchString, onlyErrors, int(page), int(size))

	searchResult, err := reader.Search(context.Background(), req)
	if err != nil {
		return
	}

	total = int64(searchResult.Aggregations().Count())
	log.Println("TOTAL: ", total)
	if total == 0 {
		return
	}

	next, err := searchResult.Next()
	for err == nil && next != nil {
		log.Println("NEXT")
		var docID string
		err = next.VisitStoredFields(func(field string, value []byte) bool {
			switch field {
			case "_id":
				fmt.Println("ID: ", string(value))
				docID = string(value)
			case "desc":
				fmt.Println("DESC: ", string(value))
			case "name":
				fmt.Println("NAME: ", string(value))
			case "tags":
				fmt.Println("TAGS: ", string(value))
			case "last_check_score":
				fmt.Println("LAST_CHECK_SCORE: ", string(value))
			default:
				fmt.Println("UNDEFINED FIELD")
			}

			return true
		})
		if err != nil {
			return
		}
		highlights := []moira.SearchHighlight{
			{
				Field: "desc",
				Value: "aaa",
			},
		}

		triggerSearchResult := moira.SearchResult{
			ObjectID:   docID,
			Highlights: highlights,
		}

		searchResults = append(searchResults, &triggerSearchResult)

		next, err = searchResult.Next()
	}
	if err != nil {
		return
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

func buildSearchRequest(filterTags []string, searchString string, onlyErrors bool, page, size int) bluge.SearchRequest {
	searchTerms := splitStringToTerms(searchString)
	searchQuery := buildSearchQuery(filterTags, searchTerms, onlyErrors)

	from := page * size
	req := bluge.NewTopNSearch(size, searchQuery).
		SetFrom(from).
		SortBy([]string{fmt.Sprintf("-%s", mapping.TriggerLastCheckScore.GetName()), "-_score", mapping.TriggerName.GetName()}).
		WithStandardAggregations()

	return req
}

func splitStringToTerms(searchString string) (searchTerms []string) {
	searchString = escapeString(searchString)

	return strings.Fields(searchString)
}

func escapeString(original string) (escaped string) {
	return regexp.MustCompile(`[|+\-=&<>!(){}\[\]^"'~*?\\/.,:;_@]`).ReplaceAllString(original, " ")
}
