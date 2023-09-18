package bluge

import (
	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/index"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/mapping"
)

func (index *TriggerIndex) Write(checks []*moira.TriggerCheck) error {
	batch := bluge.NewBatch()
	defer batch.Reset()

	for _, triggerCheck := range checks {
		if triggerCheck != nil {
			index.batchIndexTrigger(batch, triggerCheck)
		}
	}

	return index.writer.Batch(batch)
}

func (index *TriggerIndex) batchIndexTrigger(batch *index.Batch, triggerCheck *moira.TriggerCheck) {
	doc := bluge.NewDocument(triggerCheck.ID)
	indexedTrigger := mapping.CreateIndexedTrigger(triggerCheck)

	doc.AddField(bluge.NewTextField("id", indexedTrigger.ID).StoreValue())
	doc.AddField(bluge.NewTextField("desc", indexedTrigger.Desc).StoreValue().HighlightMatches())
	doc.AddField(bluge.NewTextField("name", indexedTrigger.Name).StoreValue().HighlightMatches())
	for _, tag := range triggerCheck.Tags {
		doc.AddField(bluge.NewKeywordField("tags", tag).StoreValue())
	}
	doc.AddField(bluge.NewNumericField("last_check_score", float64(indexedTrigger.LastCheckScore)).Sortable().StoreValue())

	batch.Update(doc.ID(), doc)
}
