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

			index.logger.Debugb().
				Int("triggers_count", len(newBatch)).
				Msg("Get some trigger checks from DB")

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
		index.logger.Warningb().
			String("try_number", fmt.Sprintf("%d/%d", i, triesCount)).
			Error(err).
			Msg("Cannot get trigger checks from DB")
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
				index.logger.Debugb().
					Int64("batch_size", count).
					Int("triggers_total", toIndex).
					Msg("Batch of triggers added to index")

			}(batch)
		case err, ok := <-getTriggersErrors:
			if ok {
				index.logger.ErrorWithError("Cannot get trigger checks from DB", err)
			}
			return err
		case err, ok := <-indexErrors:
			if ok {
				index.logger.ErrorWithError("Cannot index trigger checks", err)
			}
			return err
		}
	}
}
