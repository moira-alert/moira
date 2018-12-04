package index

import (
	"time"
)

func (index *Index) checkIndexedTriggersCount() error {
	checkTicker := time.NewTicker(time.Millisecond * 100)
	for {
		select {
		case <-index.tomb.Dying():
			return nil
		case <-checkTicker.C:
			if documents, err := index.index.DocCount(); err != nil {
				index.metrics.IndexedTriggersCount.Update(int64(documents))
			}
		}
	}
}

func (index *Index) checkIndexActualizationLag() error {
	checkTicker := time.NewTicker(time.Millisecond * 100)
	for {
		select {
		case <-index.tomb.Dying():
			return nil
		case <-checkTicker.C:
			index.metrics.IndexActualizationLag.UpdateSince(time.Unix(index.indexActualizedTS, 0))
		}
	}
}
