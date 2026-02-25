package bleve

import (
	"regexp"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/mapping"
)

func buildSearchQuery(options moira.SearchOptions) query.Query {
	searchQueries := make([]query.Query, 0)

	searchQueries = append(searchQueries, buildQueryForTags(options.Tags)...)
	searchQueries = append(searchQueries, buildQueryForTerms(splitStringToTerms(options.SearchString))...)
	searchQueries = append(searchQueries, buildQueryForOnlyErrors(options.OnlyProblems)...)
	searchQueries = append(searchQueries, buildQueryForCreatedBy(options.CreatedBy)...)
	searchQueries = append(searchQueries, buildQueryForTeamID(options.TeamID)...)

	if len(searchQueries) == 0 {
		return bleve.NewMatchAllQuery()
	}

	return bleve.NewConjunctionQuery(searchQueries...)
}

func buildQueryForTags(filterTags []string) (searchQueries []query.Query) {
	for _, tag := range filterTags {
		qr := bleve.NewTermQuery(tag)
		qr.FieldVal = mapping.TriggerTags.GetName()
		searchQueries = append(searchQueries, qr)
	}

	return searchQueries
}

func buildQueryForCreatedBy(createdBy string) (searchQueries []query.Query) {
	if createdBy == "" {
		return
	}

	qr := bleve.NewTermQuery(createdBy)
	qr.FieldVal = mapping.TriggerCreatedBy.GetName()
	searchQueries = append(searchQueries, qr)

	return searchQueries
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

	return searchQueries
}

func buildQueryForTeamID(teamID string) (searchQueries []query.Query) {
	if teamID == "" {
		return
	}

	qr := bleve.NewTermQuery(teamID)
	qr.FieldVal = mapping.TriggerTeamID.GetName()
	searchQueries = append(searchQueries, qr)

	return searchQueries
}

func buildQueryForOnlyErrors(onlyErrors bool) (searchQueries []query.Query) {
	if !onlyErrors {
		return searchQueries
	}

	minScore := float64(1)
	qr := bleve.NewNumericRangeQuery(&minScore, nil)
	qr.FieldVal = mapping.TriggerLastCheckScore.GetName()

	return append(searchQueries, qr)
}

func splitStringToTerms(searchString string) (searchTerms []string) {
	searchString = escapeString(searchString)

	return strings.Fields(searchString)
}

func escapeString(original string) (escaped string) {
	return regexp.MustCompile(`[|+\-=&<>!(){}\[\]^"'~*?\\/.,:;_@]`).ReplaceAllString(original, " ")
}
