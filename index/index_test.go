package index

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/blevesearch/bleve"
	"github.com/moira-alert/moira"
)

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
