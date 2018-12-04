package bleve

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
)

// TriggerIndex is implementation of index.TriggerIndex interface
type TriggerIndex struct {
	index bleve.Index
}

// CreateTriggerIndex returns TriggerIndex by provided mapping
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

// GetCount returns number of documents in TriggerIndex
func (index *TriggerIndex) GetCount() (int64, error) {
	documents, err := index.index.DocCount()
	if err != nil {
		return 0, err
	}
	return int64(documents), nil
}
