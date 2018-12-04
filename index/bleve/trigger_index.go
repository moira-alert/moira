package bleve

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
)

type TriggerIndex struct {
	index bleve.Index
}

func CreateTriggerIndex(mapping mapping.IndexMapping) (*TriggerIndex, error) {
	bleveIdx, err := bleve.NewMemOnly(mapping)
	if err != nil {
		return nil, err
	}
	newIndex := &TriggerIndex{
		index: bleveIdx,
	}
	return newIndex, nil
}

func (index *TriggerIndex) GetCount() (int64, error) {
	documents, err := index.index.DocCount()
	if err != nil {
		return 0, err
	}
	return int64(documents), nil
}
