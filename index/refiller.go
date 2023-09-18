package index

import (
	"fmt"
	"time"
)

const refillerRunInterval = 30 * time.Minute

// const batchSizeForTest = 50 // TODO: DELETE BEFORE MERGE

func (index *Index) runIndexRefiller() error {
	ticker := time.NewTicker(refillerRunInterval)
	defer ticker.Stop()
	index.logger.Info().
		Interface("refilling_interval", refillerRunInterval).
		Msg("Start refiller for search index")

	for {
		select {
		case <-index.tomb.Dying():
			index.logger.Info().Msg("Stop refilling search index")
			return nil
		case <-ticker.C:
			index.logger.Info().Msg("Refill search index by timeout")

			start := time.Now()
			if err := index.Refill(); err != nil {
				index.logger.Warning().
					Error(err).
					Msg("Cannot refill index")
				continue
			}
			end := time.Now()
			index.logger.Debug().
				Msg(fmt.Sprintf("Refill took %v sec", end.Sub(start).Seconds()))
		}
	}
}

// Completely clears the index and then repopulates it, this function is needed to clean up memory leaks that appear when updating or searching the index
func (index *Index) Refill() error {
	start := time.Now()
	triggerIds, err := index.database.GetAllTriggerIDs()
	if err != nil {
		return err
	}
	end := time.Now()
	index.logger.Debug().
		Msg(fmt.Sprintf("Fetching all trigger ids from database took %v sec", end.Sub(start).Seconds()))

	index.indexed = false
	defer func() {
		index.indexed = true
	}()
	start = time.Now()
	if err := index.deleteByBatches(triggerIds, defaultIndexBatchSize); err != nil {
		return err
	}
	end = time.Now()
	index.logger.Debug().
		Msg(fmt.Sprintf("Deleting all triggers from index took %v sec", end.Sub(start).Seconds()))

	start = time.Now()
	if err := index.fillIndex(); err != nil {
		return err
	}
	end = time.Now()
	index.logger.Debug().
		Msg(fmt.Sprintf("Filling all triggers to index took %v sec", end.Sub(start).Seconds()))

	return nil
}
