package bleve

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/index/scorch"
	"github.com/blevesearch/bleve/v2/mapping"
)

// TriggerIndex is implementation of index.TriggerIndex interface
type TriggerIndex struct {
	index bleve.Index
}

// CreateTriggerIndex returns TriggerIndex by provided mapping
func CreateTriggerIndex(mapping mapping.IndexMapping) (*TriggerIndex, error) {
	kvConfig := map[string]interface{}{
		"create_if_missing":        true,
		"error_if_exists":          true,
		"unsafe_batch":             true,
		"eventCallbackName":        "scorchEventCallbacks",
		"asyncErrorCallbackName":   "scorchAsyncErrorCallbacks",
		"numSnapshotsToKeep":       3,
		"rollbackSamplingInterval": "10m",
		"forceSegmentType":         "zap",
		"bolt_timeout":             "30s",
	}
	kvConfig["kvStoreName"] = "scorch"
	kvConfig["forceSegmentVersion"] = "7.0.0"

	bleveIdx, err := bleve.NewUsing("", mapping, scorch.Name, scorch.Name, kvConfig)
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

func (index *TriggerIndex) Close() error {
	return index.index.Close()
}
