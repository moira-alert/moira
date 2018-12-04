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
	triggerChecksChan, errorsChan := index.getTriggerChecksBatches(triggerIDsBatches)
	return index.handleTriggerBatches(triggerChecksChan, errorsChan, toIndex)
}

func (index *Index) handleTriggerBatches(triggerChecksChan chan []*moira.TriggerCheck, getTriggersErrors chan error, toIndex int) (count int64, err error) {
	indexErrors := make(chan error)
	wg := &sync.WaitGroup{}
	func() {
		for {
			select {
			case batch, ok := <-triggerChecksChan:
				if !ok {
					return
				}
				wg.Add(1)
				go func(b []*moira.TriggerCheck) {
					defer wg.Done()
					indexed, err2 := index.addBatchOfTriggerChecks(b)
					atomic.AddInt64(&count, indexed)
					if err2 != nil {
						indexErrors <- err2
						return
					}
					index.logger.Debugf("[%d triggers of %d] added to index", count, toIndex)
				}(batch)
			case err, ok := <-getTriggersErrors:
				if ok {
					index.logger.Errorf("Cannot get trigger checks from DB: %s", err.Error())
				}
				return
			case err, ok := <-indexErrors:
				if ok {
					index.logger.Errorf("Cannot index trigger checks: %s", err.Error())
				}
				return
			}
		}
	}()
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

func (index *Index) getTriggerChecksBatches(triggerIDsBatches [][]string) (triggerChecksChan chan []*moira.TriggerCheck, errors chan error) {
	wg := sync.WaitGroup{}
	triggerChecksChan = make(chan []*moira.TriggerCheck)
	errors = make(chan error)
	for _, triggerIDsBatch := range triggerIDsBatches {
		wg.Add(1)
		go func(batch []string) {
			defer wg.Done()
			newBatch, err := index.database.GetTriggerChecks(batch)
			if err != nil {
				errors <- err
				return
			}
			index.logger.Debugf("Get %d trigger checks from DB", len(newBatch))
			triggerChecksChan <- newBatch
		}(triggerIDsBatch)
	}
	go func() {
		wg.Wait()
		close(triggerChecksChan)
		close(errors)
	}()
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
