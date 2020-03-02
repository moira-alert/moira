package bleve

import (
	"github.com/blevesearch/bleve"
	"github.com/moira-alert/moira/internal/index/mapping"
	moira2 "github.com/moira-alert/moira/internal/moira"
)

// Write adds moira.TriggerChecks to TriggerIndex
func (index *TriggerIndex) Write(checks []*moira2.TriggerCheck) error {
	batch := index.index.NewBatch()
	defer batch.Reset()

	for _, trigger := range checks {
		if trigger != nil {
			err := index.batchIndexTriggerCheck(batch, trigger)
			if err != nil {
				return err
			}
		}
	}
	return index.index.Batch(batch)
}

// used as abstraction
func (index *TriggerIndex) batchIndexTriggerCheck(batch *bleve.Batch, triggerCheck *moira2.TriggerCheck) error {
	return batch.Index(triggerCheck.ID, mapping.CreateIndexedTrigger(triggerCheck))
}
