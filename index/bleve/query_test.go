package bleve

import (
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	. "github.com/smartystreets/goconvey/convey"
)

func TestIndex_BuildSearchQuery(t *testing.T) {
	tags := make([]string, 0)
	searchTerms := make([]string, 0)
	onlyErrors := false
	var createdBy string

	Convey("Test build search query", t, func() {
		Convey("Empty query", func() {
			expected := bleve.NewMatchAllQuery()
			actual := buildSearchQuery(tags, searchTerms, onlyErrors, createdBy)
			So(actual, ShouldResemble, expected)
		})

		Convey("Complex query", func() {
			Convey("Only errors = true", func() {
				onlyErrors = true
				qr := buildQueryForOnlyErrors(onlyErrors)
				expected := bleve.NewConjunctionQuery(qr...)
				actual := buildSearchQuery(tags, searchTerms, onlyErrors, createdBy)
				So(actual, ShouldResemble, expected)
			})

			Convey("Only errors = false, several tags", func() {
				onlyErrors = false
				tags = []string{"123", "456"}
				qr := buildQueryForTags(tags)
				expected := bleve.NewConjunctionQuery(qr...)
				actual := buildSearchQuery(tags, searchTerms, onlyErrors, createdBy)
				So(actual, ShouldResemble, expected)
			})

			Convey("Only errors = false, no tags, several terms", func() {
				onlyErrors = false
				tags = make([]string, 0)
				searchTerms = []string{"123", "456"}
				qr := buildQueryForTerms(searchTerms)
				expected := bleve.NewConjunctionQuery(qr...)
				actual := buildSearchQuery(tags, searchTerms, onlyErrors, createdBy)
				So(actual, ShouldResemble, expected)
			})

			Convey("Only errors = false, several tags, several terms", func() {
				onlyErrors = false
				tags = []string{"123", "456"}
				searchTerms = []string{"123", "456"}
				searchQueries := make([]query.Query, 0)

				searchQueries = append(searchQueries, buildQueryForTags(tags)...)
				searchQueries = append(searchQueries, buildQueryForTerms(searchTerms)...)
				searchQueries = append(searchQueries, buildQueryForOnlyErrors(onlyErrors)...)
				expected := bleve.NewConjunctionQuery(searchQueries...)

				actual := buildSearchQuery(tags, searchTerms, onlyErrors, createdBy)
				So(actual, ShouldResemble, expected)
			})

			Convey("Only errors = false, no tags, without terms, with created by", func() {
				onlyErrors = false
				tags = make([]string, 0)
				searchTerms = make([]string, 0)
				createdBy = "test"
				qr := buildQueryForCreatedBy(createdBy)
				expected := bleve.NewConjunctionQuery(qr...)

				actual := buildSearchQuery(tags, searchTerms, onlyErrors, createdBy)
				So(actual, ShouldResemble, expected)
			})

			Convey("Only errors = true, several tags, several terms, with created by", func() {
				onlyErrors = true
				tags = []string{"123", "456"}
				searchTerms = []string{"123", "456"}
				searchQueries := make([]query.Query, 0)

				searchQueries = append(searchQueries, buildQueryForTags(tags)...)
				searchQueries = append(searchQueries, buildQueryForTerms(searchTerms)...)
				searchQueries = append(searchQueries, buildQueryForOnlyErrors(onlyErrors)...)
				searchQueries = append(searchQueries, buildQueryForCreatedBy(createdBy)...)
				expected := bleve.NewConjunctionQuery(searchQueries...)

				actual := buildSearchQuery(tags, searchTerms, onlyErrors, createdBy)
				So(actual, ShouldResemble, expected)
			})
		})
	})
}
