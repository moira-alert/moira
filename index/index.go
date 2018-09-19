package main

import (
	"fmt"

	"github.com/blevesearch/bleve"
	"github.com/moira-alert/moira"
)

func getIndex(indexPath string) (bleve.Index, error) {

	bleveIdx, err := bleve.Open(indexPath)
	if err != nil {
		mapping := bleve.NewIndexMapping()
		bleveIdx, err = bleve.New(indexPath, mapping)
		if err != nil {
			return nil, err
		}
	}

	// return de index
	return bleveIdx, nil
}

func main() {
	bleveIdx, err := getIndex("moira.example")

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
	}

	// index some data
	bleveIdx.Index("id", data)

	// search for some text
	query := bleve.NewMatchQuery("words")
	search := bleve.NewSearchRequest(query)
	searchResults, err := bleveIdx.Search(search)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(searchResults)
}
