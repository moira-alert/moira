package bluge

import (
	"github.com/blugelabs/bluge"
)

func (index *TriggerIndex) Delete(triggerIDs []string) error {
	batch := bluge.NewBatch()
	defer batch.Reset()

	for _, triggerID := range triggerIDs {
		doc := bluge.NewDocument(triggerID)
		batch.Delete(doc.ID())
	}

	return index.writer.Batch(batch)
}
