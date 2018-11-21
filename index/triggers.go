package index

import (
	"github.com/blevesearch/bleve"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/mapping"
)

func (index *Index) fillIndex() error {
	index.logger.Debugf("Start filling index with triggers")
	index.inProgress = true
	allTriggerIDs, err := index.database.GetAllTriggerIDs()
	index.logger.Debugf("Triggers IDs fetched from database: %d", len(allTriggerIDs))
	if err != nil {
		return err
	}

	count, err := index.addTriggers(allTriggerIDs, indexBatchSize)
	index.logger.Infof("%d triggers added to index", count)
	return err
}

func (index *Index) addTriggers(triggerIDs []string, batchSize int) (count int, err error) {
	toIndex := len(triggerIDs)
	batch := index.index.NewBatch()
	firstIndexed := false

	triggerIDsBatches := moira.ChunkSlice(triggerIDs, batchSize)
	var triggersToCheck []*moira.TriggerCheck

	for _, triggerIDsBatch := range triggerIDsBatches {
		triggersToCheck, err = index.database.GetTriggerChecks(triggerIDsBatch)
		index.logger.Debugf("Get %d trigger checks from DB", len(triggersToCheck))
		if err != nil {
			return
		}
		for _, trigger := range triggersToCheck {
			if trigger != nil {
				// ToDo: this code works, but looks stupid. We have to find a reason why Bleve indexes first batch 1 minute
				if !firstIndexed {
					index.indexTriggerCheck(trigger)
					firstIndexed = true
				}
				err = index.batchIndexTriggerCheck(batch, trigger)
				if err != nil {
					return
				}
			}
		}
		err = index.index.Batch(batch)
		if err != nil {
			return
		}
		count += batch.Size()
		batch.Reset()
		index.logger.Debugf("[%d triggers of %d] added to index", count, toIndex)
	}
	return
}

// used as abstraction
func (index *Index) indexTriggerCheck(triggerCheck *moira.TriggerCheck) error {
	return index.index.Index(triggerCheck.ID, mapping.CreateIndexedTrigger(triggerCheck))
}

// used as abstraction
func (index *Index) batchIndexTriggerCheck(batch *bleve.Batch, triggerCheck *moira.TriggerCheck) error {
	return batch.Index(triggerCheck.ID, mapping.CreateIndexedTrigger(triggerCheck))
}
