package main

import (
	"fmt"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/analysis/lang/en"
	"github.com/blevesearch/bleve/mapping"
	"github.com/moira-alert/moira"
)

const indexName = "moira.example"

func getIndex(indexPath string) (bleve.Index, error) {

	bleveIdx, err := bleve.Open(indexPath)
	if err != nil {
		indexMapping := buildIndexMapping()
		bleveIdx, err = bleve.New(indexPath, indexMapping)
		if err != nil {
			return nil, err
		}
	}

	// return de index
	return bleveIdx, nil
}

func buildIndexMapping() mapping.IndexMapping {

	// a generic reusable mapping for english text
	englishTextFieldMapping := bleve.NewTextFieldMapping()
	englishTextFieldMapping.Analyzer = en.AnalyzerName

	// a generic reusable mapping for keyword text
	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Analyzer = keyword.Name

	triggerMapping := bleve.NewDocumentMapping()

	triggerMapping.AddFieldMappingsAt("name", englishTextFieldMapping)
	triggerMapping.AddFieldMappingsAt("description", englishTextFieldMapping)
	triggerMapping.AddFieldMappingsAt("tags", keywordFieldMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping(moira.Trigger{}.Type(), triggerMapping)
	return indexMapping
}

func main() {
	bleveIdx, err := getIndex(indexName)

	desc := "Test trigger description. Many words are written here. None of them are useful."
	expr := "OK"

	data := moira.Trigger{
		ID:   "TestTriggerID",
		Name: "Test trigger name",
		Desc: &desc,
		Targets: []string{
			"Super.Awesome.Metrics.*",
		},
		TriggerType: "expression",
		Expression:  &expr,
		Tags: []string{
			"DevOps", "critical", "awesome-tag"},
	}

	// index some data
	bleveIdx.Index(data.ID, data)

	// search for some text
	query := bleve.NewMatchQuery("expression")
	search := bleve.NewSearchRequest(query)
	searchResults, err := bleveIdx.Search(search)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(searchResults)
}
