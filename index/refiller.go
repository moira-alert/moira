package index

import "time"

const refillerRunInterval = 15 * time.Minute
const batchSizeForTest = 50 // TODO: DELETE BEFORE MERGE

func (index *Index) runIndexRefiller() error {
	ticker := time.NewTicker(refillerRunInterval)
	defer ticker.Stop()
	index.logger.Info().
		Interface("refilling_interval", refillerRunInterval).
		Msg("Start refilling search index")

	for {
		select {
		case <-index.tomb.Dying():
			index.logger.Info().Msg("Stop refilling search index")
			return nil
		case <-ticker.C:
			index.logger.Info().Msg("Refill search index by timeout")

			if err := index.Refill(); err != nil {
				index.logger.Warning().
					Error(err).
					Msg("Cannot refill index")
				continue
			}
		}
	}
}

// Completely clears the index and then repopulates it, this function is needed to clean up memory leaks that appear when updating or searching the index
func (index *Index) Refill() error {
	triggerIds, err := index.database.GetAllTriggerIDs()
	if err != nil {
		return err
	}

	index.indexed = false
	defer func() {
		index.indexed = true
	}()
	if err := index.deleteByBatches(triggerIds, batchSizeForTest); err != nil {
		return err
	}
	// if err := index.triggerIndex.Delete(triggerIds); err != nil {
	// 	return err
	// }
	if err := index.fillIndex(); err != nil {
		return err
	}

	return nil
}
