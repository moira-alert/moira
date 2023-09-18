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
	return index.writeTriggerBatches(triggerChecksChan, errorsChan, len(triggerIDs))
}

func (index *Index) deleteByBatches(triggerIDs []string, batchSize int) error {
	triggerIDsBatches := moira.ChunkSlice(triggerIDs, batchSize)
	triggerIDsChan := getChannelWithTriggerIDsBatches(triggerIDsBatches)
	return index.deleteTriggerBatches(triggerIDsChan, len(triggerIDs))
}

func getChannelWithTriggerIDsBatches(triggerIDsBatches [][]string) <-chan []string {
	ch := make(chan []string, len(triggerIDsBatches))

	go func() {
		defer close(ch)

		for _, triggerIDsBatch := range triggerIDsBatches {
			ch <- triggerIDsBatch
		}
	}()

	return ch
}

func (index *Index) deleteTriggerBatches(triggerIDsChan <-chan []string, triggersTotal int) error {
	indexErrors := make(chan error)
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	var count int64

	for {
		select {
		case batch, ok := <-triggerIDsChan:
			if !ok {
				return nil
			}

			wg.Add(1)
			go func(b []string) {
				defer wg.Done()
				err := index.triggerIndex.Delete(b)
				atomic.AddInt64(&count, int64(len(b)))
				if err != nil {
					indexErrors <- err
					return
				}
				index.logger.Debug().
					Int("batch_size", len(batch)).
					Int64("count", atomic.LoadInt64(&count)).
					Int("triggers_total", triggersTotal).
					Msg("Batch of triggers deleted from index")
			}(batch)

		case err, ok := <-indexErrors:
			if ok {
				index.logger.Error().
					Error(err).
					Msg("Cannot index trigger checks")
			}
			return err
		}
	}
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

			index.logger.Debug().
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
		index.logger.Warning().
			String("try_count", fmt.Sprintf("%d/%d", i, triesCount)).
			Error(err).
			Msg("Cannot get trigger checks from DB")
	}
	return nil, fmt.Errorf("cannot get trigger checks from DB after %d tries, last error: %s", triesCount, err.Error())
}

func (index *Index) writeTriggerBatches(triggerChecksChan chan []*moira.TriggerCheck, getTriggersErrors chan error, triggersTotal int) error {
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
				err := index.triggerIndex.Write(b)
				atomic.AddInt64(&count, int64(len(b)))
				if err != nil {
					indexErrors <- err
					return
				}
				index.logger.Debug().
					Int("batch_size", len(batch)).
					Int64("count", atomic.LoadInt64(&count)).
					Int("triggers_total", triggersTotal).
					Msg("Batch of triggers added to index")
			}(batch)
		case err, ok := <-getTriggersErrors:
			if ok {
				index.logger.Error().
					Error(err).
					Msg("Cannot get trigger checks from DB")
			}
			return err
		case err, ok := <-indexErrors:
			if ok {
				index.logger.Error().
					Error(err).
					Msg("Cannot index trigger checks")
			}
			return err
		}
	}
}
