package bleve

// Delete removes triggerIDs from TriggerIndex
func (index *TriggerIndex) Delete(triggerIDs []string) error {
	batch := index.index.NewBatch()
	defer batch.Reset()

	for _, triggerID := range triggerIDs {
		batch.Delete(triggerID)
	}
	return index.index.Batch(batch)
}
