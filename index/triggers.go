package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/mapping"
)

var (
	fakeTriggerToIndex = &moira.TriggerCheck{
		Trigger: moira.Trigger{
			ID:   "This.Is.Fake.Trigger.ID.It.Should.Not.Exist.In.Real.Life",
			Name: "Fake trigger to index",
		},
		LastCheck: moira.CheckData{
			Score: 0,
		},
	}
)

func (index *Index) fillIndex() error {
	index.logger.Debugf("Start filling index with triggers")
	index.inProgress = true
	index.indexActualizedTS = time.Now().Unix()
	allTriggerIDs, err := index.database.GetAllTriggerIDs()
	index.logger.Debugf("Triggers IDs fetched from database: %d", len(allTriggerIDs))
	if err != nil {
		return err
	}

	count, err := index.addTriggers(allTriggerIDs, defaultIndexBatchSize)
	index.logger.Infof("%d triggers added to index", count)
	return err
}

func (index *Index) addTriggers(triggerIDs []string, batchSize int) (count int, err error) {
	toIndex := len(triggerIDs)

	if !index.indexed {
		// We index fake trigger to increase batch index speed. Otherwise, first batch is indexed for too long
		index.indexTriggerCheck(fakeTriggerToIndex)
		defer index.index.Delete(fakeTriggerToIndex.ID)
	}

	triggerIDsBatches := moira.ChunkSlice(triggerIDs, batchSize)
	var triggerChecksToIndex []*moira.TriggerCheck

	for _, triggerIDsBatch := range triggerIDsBatches {
		var indexed int
		triggerChecksToIndex, err = index.database.GetTriggerChecks(triggerIDsBatch)
		index.logger.Debugf("Get %d trigger checks from DB", len(triggerChecksToIndex))
		if err != nil {
			return
		}
		indexed, err = index.addBatchOfTriggerChecks(triggerChecksToIndex)
		count = count + indexed
		if err != nil {
			return
		}
		index.logger.Debugf("[%d triggers of %d] added to index", count, toIndex)
	}
	return
}

func (index *Index) addBatchOfTriggerChecks(triggerChecks []*moira.TriggerCheck) (count int, err error) {
	batch := index.index.NewBatch()
	defer batch.Reset()

	for _, trigger := range triggerChecks {
		if trigger != nil {
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
