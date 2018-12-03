package index

import (
	"sync"
	"sync/atomic"
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

func (index *Index) addTriggers(triggerIDs []string, batchSize int) (count int64, err error) {
	toIndex := len(triggerIDs)
	if !index.indexed {
		// We index fake trigger to increase batch index speed. Otherwise, first batch is indexed for too long
		index.indexTriggerCheck(fakeTriggerToIndex)
		defer index.index.Delete(fakeTriggerToIndex.ID)
	}

	triggerIDsBatches := moira.ChunkSlice(triggerIDs, batchSize)

	triggerChecksChan := make(chan []*moira.TriggerCheck)
	errorsChan := make(chan error)
	go index.getTriggerChecksBatches(triggerIDsBatches, triggerChecksChan, errorsChan)

	wg := &sync.WaitGroup{}

Loop:
	for range triggerIDs {
		select {
		case batch, ok := <-triggerChecksChan:
			if !ok {
				break Loop
			}
			index.logger.Debugf("Get %d trigger checks from DB", len(batch))
			wg.Add(1)
			go func(b []*moira.TriggerCheck) {
				defer wg.Done()
				indexed, err := index.addBatchOfTriggerChecks(b)
				atomic.AddInt64(&count, indexed)
				if err != nil {
					return
				}
				index.logger.Debugf("[%d triggers of %d] added to index", count, toIndex)
			}(batch)
		case err := <-errorsChan:
			index.logger.Errorf("Cannot get trigger checks from DB: %s", err.Error())
		}
	}
	wg.Wait()
	return
}

func (index *Index) addBatchOfTriggerChecks(triggerChecks []*moira.TriggerCheck) (count int64, err error) {
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
	count += int64(batch.Size())
	return
}

func (index *Index) getTriggerChecksBatches(triggerIDsBatches [][]string, triggerChecksChan chan []*moira.TriggerCheck, errors chan error) {
	for _, triggerIDsBatch := range triggerIDsBatches {
		newBatch, err := index.database.GetTriggerChecks(triggerIDsBatch)
		if err != nil {
			errors <- err
			return
		}
		triggerChecksChan <- newBatch
	}
	close(triggerChecksChan)
}

// used as abstraction
func (index *Index) indexTriggerCheck(triggerCheck *moira.TriggerCheck) error {
	return index.index.Index(triggerCheck.ID, mapping.CreateIndexedTrigger(triggerCheck))
}

// used as abstraction
func (index *Index) batchIndexTriggerCheck(batch *bleve.Batch, triggerCheck *moira.TriggerCheck) error {
	return batch.Index(triggerCheck.ID, mapping.CreateIndexedTrigger(triggerCheck))
}
