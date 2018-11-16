package index

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/moira-alert/moira/index/mapping"
)

func buildSearchQuery(filterTags, searchTerms []string, onlyErrors bool) query.Query {
	if !onlyErrors && len(filterTags) == 0 && len(searchTerms) == 0 {
		return bleve.NewMatchAllQuery()
	}

	searchQueries := make([]query.Query, 0)

	searchQueries = append(searchQueries, buildQueryForTags(filterTags)...)
	searchQueries = append(searchQueries, buildQueryForTerms(searchTerms)...)
	searchQueries = append(searchQueries, buildQueryForOnlyErrors(onlyErrors)...)

	return bleve.NewConjunctionQuery(searchQueries...)
}

func buildQueryForTags(filterTags []string) (searchQueries []query.Query) {
	for _, tag := range filterTags {
		qr := bleve.NewTermQuery(tag)
		qr.FieldVal = mapping.TriggerTags.String()
		searchQueries = append(searchQueries, qr)
	}
	return
}

func buildQueryForTerms(searchTerms []string) (searchQueries []query.Query) {
	for _, term := range searchTerms {
		qr := bleve.NewFuzzyQuery(term)
		searchQueries = append(searchQueries, qr)
	}
	return
}

func buildQueryForOnlyErrors(onlyErrors bool) (searchQueries []query.Query) {
	if !onlyErrors {
		return
	}
	minScore := float64(1)
	qr := bleve.NewNumericRangeQuery(&minScore, nil)
	qr.FieldVal = mapping.TriggerLastCheckScore.String()
	return append(searchQueries, qr)
}
