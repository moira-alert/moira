package bleve

import (
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
)

const defaultSearchString = "123 456"

func TestIndex_BuildSearchQuery(t *testing.T) {
	searchOptions := moira.SearchOptions{
		OnlyProblems:          false,
		Tags:                  make([]string, 0),
		SearchString:          "",
		CreatedBy:             "",
		NeedSearchByCreatedBy: false,
	}

	Convey("Test build search query", t, func() {
		Convey("Empty query", func() {
			expected := bleve.NewMatchAllQuery()
			actual := buildSearchQuery(searchOptions)
			So(actual, ShouldResemble, expected)
		})

		Convey("Complex query", func() {
			Convey("Only errors = true", func() {
				searchOptions.OnlyProblems = true
				qr := buildQueryForOnlyErrors(searchOptions.OnlyProblems)
				expected := bleve.NewConjunctionQuery(qr...)
				actual := buildSearchQuery(searchOptions)
				So(actual, ShouldResemble, expected)
			})

			Convey("Only errors = false, several tags", func() {
				searchOptions.OnlyProblems = false
				searchOptions.Tags = []string{"123", "456"}
				qr := buildQueryForTags(searchOptions.Tags)
				expected := bleve.NewConjunctionQuery(qr...)
				actual := buildSearchQuery(searchOptions)
				So(actual, ShouldResemble, expected)
			})

			Convey("Only errors = false, no tags, several terms", func() {
				searchOptions.OnlyProblems = false
				searchOptions.Tags = make([]string, 0)
				searchOptions.SearchString = defaultSearchString
				searchTerms := []string{"123", "456"}

				qr := buildQueryForTerms(searchTerms)
				expected := bleve.NewConjunctionQuery(qr...)
				actual := buildSearchQuery(searchOptions)
				So(actual, ShouldResemble, expected)
			})

			Convey("Only errors = false, several tags, several terms", func() {
				searchOptions.OnlyProblems = false
				searchOptions.Tags = []string{"123", "456"}
				searchOptions.SearchString = defaultSearchString
				searchTerms := []string{"123", "456"}

				searchQueries := make([]query.Query, 0)

				searchQueries = append(searchQueries, buildQueryForTags(searchOptions.Tags)...)
				searchQueries = append(searchQueries, buildQueryForTerms(searchTerms)...)
				searchQueries = append(searchQueries, buildQueryForOnlyErrors(searchOptions.OnlyProblems)...)
				expected := bleve.NewConjunctionQuery(searchQueries...)

				actual := buildSearchQuery(searchOptions)
				So(actual, ShouldResemble, expected)
			})

			Convey("Only errors = false, no tags, without terms, with created by", func() {
				searchOptions.OnlyProblems = false
				searchOptions.Tags = make([]string, 0)
				searchOptions.SearchString = ""
				searchOptions.CreatedBy = "test"
				searchOptions.NeedSearchByCreatedBy = true

				qr := buildQueryForCreatedBy(searchOptions.CreatedBy, searchOptions.NeedSearchByCreatedBy)
				expected := bleve.NewConjunctionQuery(qr...)

				actual := buildSearchQuery(searchOptions)
				So(actual, ShouldResemble, expected)
			})

			Convey("Only errors = true, several tags, several terms, with created by", func() {
				searchOptions.OnlyProblems = true
				searchOptions.Tags = []string{"123", "456"}
				searchOptions.SearchString = defaultSearchString
				searchTerms := []string{"123", "456"}

				searchQueries := make([]query.Query, 0)

				searchQueries = append(searchQueries, buildQueryForTags(searchOptions.Tags)...)
				searchQueries = append(searchQueries, buildQueryForTerms(searchTerms)...)
				searchQueries = append(searchQueries, buildQueryForOnlyErrors(searchOptions.OnlyProblems)...)
				searchQueries = append(searchQueries, buildQueryForCreatedBy(searchOptions.CreatedBy, searchOptions.NeedSearchByCreatedBy)...)
				expected := bleve.NewConjunctionQuery(searchQueries...)

				actual := buildSearchQuery(searchOptions)
				So(actual, ShouldResemble, expected)
			})
		})
	})
}
