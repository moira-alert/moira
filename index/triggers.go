package index

import (
	"time"

	"github.com/moira-alert/moira"
)

var (
	fakeTriggerToIndex = &moira.TriggerCheck{
		Trigger: moira.Trigger{
			ID:   "This.Is.Fake.Trigger.ID.It.Should.Not.Exist.In.Real.Life",
			Name: "Fake trigger to index",
		},
		LastCheck: moira.CheckData{
			Score: 0,
		},
	}
)

func (index *Index) fillIndex() error {
	index.logger.Debugb().Msg("Start filling index with triggers")

	index.inProgress = true
	index.indexActualizedTS = time.Now().Unix()
	allTriggerIDs, err := index.database.GetAllTriggerIDs()

	index.logger.Debugb().
		Int("Quantity", len(allTriggerIDs)).
		Msg("Triggers IDs fetched from database")

	if err != nil {
		return err
	}

	// We index fake trigger to increase batch index speed. Otherwise, first batch is indexed for too long
	index.triggerIndex.Write([]*moira.TriggerCheck{fakeTriggerToIndex}) //nolint
	defer index.triggerIndex.Delete([]string{fakeTriggerToIndex.ID})    //nolint

	err = index.writeByBatches(allTriggerIDs, defaultIndexBatchSize)

	index.logger.Infob().
		Int("Quantity", len(allTriggerIDs)).
		Msg("Added triggers to index")

	return err
}
