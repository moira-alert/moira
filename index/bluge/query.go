package bluge

import (
	"github.com/blugelabs/bluge"
	"github.com/moira-alert/moira/index/mapping"
)

func buildSearchQuery(filterTags, searchTerms []string, onlyErrors bool) bluge.Query {
	if !onlyErrors && len(filterTags) == 0 && len(searchTerms) == 0 {
		return bluge.NewMatchAllQuery()
	}

	searchQuery := bluge.NewBooleanQuery()
	searchQuery.AddMust(buildQueryForTags(filterTags))
	searchQuery.AddMust(buildQueryForTerms(searchTerms))
	searchQuery.AddMust(buildQueryForOnlyErrors(onlyErrors))

	return searchQuery
}

func buildQueryForTags(filterTags []string) bluge.Query {
	searchQuery := bluge.NewBooleanQuery()
	for _, tag := range filterTags {
		query := bluge.NewTermQuery(tag)
		query.SetField(mapping.TriggerTags.GetName())
		searchQuery.AddShould(query)
	}
	return searchQuery
}

func buildQueryForTerms(searchTerms []string) bluge.Query {
	searchQuery := bluge.NewBooleanQuery()
	for _, term := range searchTerms {
		nameQuery, nameField := bluge.NewFuzzyQuery(term), mapping.TriggerName
		nameQuery.SetField(nameField.GetName())
		nameQuery.SetBoost(nameField.GetPriority())

		descQuery, descField := bluge.NewFuzzyQuery(term), mapping.TriggerDesc
		descQuery.SetField(descField.GetName())
		descQuery.SetBoost(descField.GetPriority())

		boolQuery := bluge.NewBooleanQuery()
		boolQuery.AddShould(nameQuery)
		boolQuery.AddShould(descQuery)

		searchQuery.AddShould(boolQuery)
	}
	return searchQuery
}

func buildQueryForOnlyErrors(onlyErrors bool) bluge.Query {
	searchQuery := bluge.NewBooleanQuery()
	if !onlyErrors {
		return searchQuery
	}
	minScore := 1.0
	maxScore := 10.0
	query := bluge.NewNumericRangeQuery(minScore, maxScore)
	query.SetField(mapping.TriggerLastCheckScore.GetName())
	return searchQuery.AddMust(query)
}
