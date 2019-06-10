package bleve

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
		qr.SetField(mapping.TriggerName.String())
		qr.SetBoost(3)
		qr1 := bleve.NewFuzzyQuery(term)
		qr1.SetField(mapping.TriggerDesc.String())
		qr1.SetBoost(1)
		searchQueries = append(searchQueries, bleve.NewDisjunctionQuery(qr, qr1))
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
