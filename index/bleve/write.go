package bleve

import (
	"log"

	"github.com/blevesearch/bleve/v2"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/mapping"
)

// Write adds moira.TriggerChecks to TriggerIndex
func (index *TriggerIndex) Write(checks []*moira.TriggerCheck) error {
	stats, _ := index.index.Stats().MarshalJSON()
	log.Println("INDEX STATS: ", string(stats))

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
func (index *TriggerIndex) batchIndexTriggerCheck(batch *bleve.Batch, triggerCheck *moira.TriggerCheck) error {
	return batch.Index(triggerCheck.ID, mapping.CreateIndexedTrigger(triggerCheck))
}
