package bleve

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/moira-alert/moira/index/mapping"
)

func buildSearchQuery(filterTags, searchTerms []string, onlyErrors, needSearchByCreatedBy bool, createdBy string) query.Query {
	if !onlyErrors && len(filterTags) == 0 && len(searchTerms) == 0 && !needSearchByCreatedBy {
		return bleve.NewMatchAllQuery()
	}

	searchQueries := make([]query.Query, 0)

	searchQueries = append(searchQueries, buildQueryForTags(filterTags)...)
	searchQueries = append(searchQueries, buildQueryForTerms(searchTerms)...)
	searchQueries = append(searchQueries, buildQueryForOnlyErrors(onlyErrors)...)
	searchQueries = append(searchQueries, buildQueryForCreatedBy(createdBy, needSearchByCreatedBy)...)

	return bleve.NewConjunctionQuery(searchQueries...)
}

func buildQueryForTags(filterTags []string) (searchQueries []query.Query) {
	for _, tag := range filterTags {
		qr := bleve.NewTermQuery(tag)
		qr.FieldVal = mapping.TriggerTags.GetName()
		searchQueries = append(searchQueries, qr)
	}
	return
}

func buildQueryForCreatedBy(createdBy string, needSearchByCreatedBy bool) (searchQueries []query.Query) {
	if !needSearchByCreatedBy {
		return
	}
	qr := bleve.NewTermQuery(createdBy)
	qr.FieldVal = mapping.TriggerCreatedBy.GetName()
	searchQueries = append(searchQueries, qr)
	return
}

func buildQueryForTerms(searchTerms []string) (searchQueries []query.Query) {
	for _, term := range searchTerms {
		nameQuery, nameField := bleve.NewFuzzyQuery(term), mapping.TriggerName
		nameQuery.SetField(nameField.GetName())
		nameQuery.SetBoost(nameField.GetPriority())
		descQuery, descField := bleve.NewFuzzyQuery(term), mapping.TriggerDesc
		descQuery.SetField(descField.GetName())
		descQuery.SetBoost(descField.GetPriority())
		searchQueries = append(searchQueries, bleve.NewDisjunctionQuery(nameQuery, descQuery))
	}
	return
}

func buildQueryForOnlyErrors(onlyErrors bool) (searchQueries []query.Query) {
	if !onlyErrors {
		return
	}
	minScore := float64(1)
	qr := bleve.NewNumericRangeQuery(&minScore, nil)
	qr.FieldVal = mapping.TriggerLastCheckScore.GetName()
	return append(searchQueries, qr)
}
