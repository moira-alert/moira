package index

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/moira-alert/moira"
)

func (index *Index) writeByBatches(triggerIDs []string, batchSize int) error {
	triggerIDsBatches := moira.ChunkSlice(triggerIDs, batchSize)
	triggerChecksChan, errorsChan := index.getTriggerChecksBatches(triggerIDsBatches)
	return index.handleTriggerBatches(triggerChecksChan, errorsChan, len(triggerIDs))
}

func (index *Index) getTriggerChecksBatches(triggerIDsBatches [][]string) (triggerChecksChan chan []*moira.TriggerCheck, errors chan error) {
	wg := sync.WaitGroup{}
	triggerChecksChan = make(chan []*moira.TriggerCheck)
	errors = make(chan error)
	for _, triggerIDsBatch := range triggerIDsBatches {
		wg.Add(1)
		go func(batch []string) {
			defer wg.Done()

			newBatch, err := index.getTriggerChecksWithRetries(batch)
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

func (index *Index) getTriggerChecksWithRetries(batch []string) ([]*moira.TriggerCheck, error) {
	var err error
	triesCount := 3
	for i := 1; i <= triesCount; i++ {
		var newBatch []*moira.TriggerCheck
		newBatch, err = index.database.GetTriggerChecks(batch)
		if err == nil {
			return newBatch, nil
		}
		index.logger.Warningf("Cannot get trigger checks from DB, try %d/%d: %s", i, triesCount, err.Error())
	}
	return nil, fmt.Errorf("cannot get trigger checks from DB after %d tries, last error: %s", triesCount, err.Error())
}

func (index *Index) handleTriggerBatches(triggerChecksChan chan []*moira.TriggerCheck, getTriggersErrors chan error, toIndex int) error {
	indexErrors := make(chan error)
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	var count int64

	for {
		select {
		case batch, ok := <-triggerChecksChan:
			if !ok {
				return nil
			}
			wg.Add(1)
			go func(b []*moira.TriggerCheck) {
				defer wg.Done()
				err2 := index.triggerIndex.Write(b)
				atomic.AddInt64(&count, int64(len(b)))
				if err2 != nil {
					indexErrors <- err2
					return
				}
				index.logger.Debugf("[%d triggers of %d] added to index", atomic.LoadInt64(&count), toIndex)
			}(batch)
		case err, ok := <-getTriggersErrors:
			if ok {
				index.logger.Errorf("Cannot get trigger checks from DB: %s", err.Error())
			}
			return err
		case err, ok := <-indexErrors:
			if ok {
				index.logger.Errorf("Cannot index trigger checks: %s", err.Error())
			}
			return err
		}
	}
}
