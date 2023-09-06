package index

import (
	"time"
)

func (index *Index) checkIndexedTriggersCount() error {
	checkTicker := time.NewTicker(time.Millisecond * 100) //nolint
	defer checkTicker.Stop()

	for {
		select {
		case <-index.tomb.Dying():
			return nil
		case <-checkTicker.C:
			if !index.checkIfIndexIsReady() {
				index.logger.Warning().
					Msg("Cannot check indexed triggers count cause index is not ready")
				continue
			}
			if documents, err := index.triggerIndex.GetCount(); err == nil {
				index.metrics.IndexedTriggersCount.Update(documents)
			}
		}
	}
}

func (index *Index) checkIndexActualizationLag() error {
	checkTicker := time.NewTicker(time.Millisecond * 100) //nolint
	defer checkTicker.Stop()

	for {
		select {
		case <-index.tomb.Dying():
			return nil
		case <-checkTicker.C:
			if !index.checkIfIndexIsReady() {
				index.logger.Warning().
					Msg("Cannot check index actualization lag cause index is not ready")
				continue
			}
			index.metrics.IndexActualizationLag.UpdateSince(time.Unix(index.indexActualizedTS, 0))
		}
	}
}
