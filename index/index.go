package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/mapping"
	"github.com/moira-alert/moira"
)

const indexName = "moira.example"

type listOfTriggers struct {
	List []moira.Trigger `json:"list"`
}

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
	standardFieldMapping := bleve.NewTextFieldMapping()
	standardFieldMapping.Analyzer = standard.Name

	// a generic reusable mapping for keyword text
	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Analyzer = keyword.Name

	triggerMapping := bleve.NewDocumentMapping()

	triggerMapping.AddFieldMappingsAt("name", standardFieldMapping)
	triggerMapping.AddFieldMappingsAt("description", standardFieldMapping)
	triggerMapping.AddFieldMappingsAt("tags", keywordFieldMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping(moira.Trigger{}.Type(), triggerMapping)
	return indexMapping
}

func loadTriggers(fileName string) ([]moira.Trigger, error) {
	jsonFile, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var triggers listOfTriggers
	err = json.Unmarshal(byteValue, &triggers)
	return triggers.List, err
}

func indexTriggers(triggers []moira.Trigger, index bleve.Index) error {
	// ToDo: make it batch
	for _, tr := range triggers {
		err := index.Index(tr.ID, tr)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	bleveIdx, err := getIndex(indexName)

	triggers, err := loadTriggers("index\\triggers_test_data.json")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = indexTriggers(triggers, bleveIdx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// search for some text
	//qString := `+tags:DevOps`
	//qString += ` +tags:normal`
	//qString += ` -desc:тут`
	tq1 := bleve.NewFuzzyQuery("trigger")
	tq2 := bleve.NewTermQuery("Moira")
	tq2.FieldVal = "tags"
	qr := bleve.NewConjunctionQuery(tq1, tq2)
	req := bleve.NewSearchRequest(qr)
	req.Fields = []string{"id", "name", "tags", "desc"}
	req.Highlight = bleve.NewHighlightWithStyle("html")
	req.Explain = true
	searchResults, err := bleveIdx.Search(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(searchResults)
}
