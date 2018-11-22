package index

import (
	"time"
)

const actualizerRunInterval = time.Second

func (index *Index) runIndexActualizer() error {
	ticker := time.NewTicker(actualizerRunInterval)
	index.logger.Infof("Start index actualizer: reindex changed triggers every %v", actualizerRunInterval)

	for {
		select {
		case <-index.tomb.Dying():
			index.logger.Info("Stop index actualizer")
			return nil
		case <-ticker.C:
			newTime := time.Now().Unix()
			if float64(newTime-index.indexActualizedTS) > sweeperTimeToKeep.Seconds() {
				index.logger.Errorf("Index was actualized too far ago. Index actualized: %s. Current time: %s. Should actualize every: %v. Maximum possible interval without actualization: %s",
					time.Unix(index.indexActualizedTS, 0).Format(time.RFC3339), time.Now().Format(time.RFC3339), actualizerRunInterval, sweeperTimeToKeep)
			}
			if err := index.actualizeIndex(); err != nil {
				index.logger.Warningf("Cannot actualize triggers: %s", err.Error())
				continue
			}
			index.indexActualizedTS = newTime
		}
	}
}

func (index *Index) actualizeIndex() error {
	triggerToReindexIDs, err := index.database.FetchTriggersToReindex(index.indexActualizedTS)
	if err != nil {
		return err
	}

	if len(triggerToReindexIDs) == 0 {
		return nil
	}

	index.logger.Debugf("[Index actualizer]: got %d triggers to actualize", len(triggerToReindexIDs))

	triggersToReindex, err := index.database.GetTriggerChecks(triggerToReindexIDs)
	if err != nil {
		return err
	}
	triggersToUpdate := make([]string, 0)

	for i, triggerID := range triggerToReindexIDs {
		trigger := triggersToReindex[i]
		if trigger == nil {
			index.logger.Debugf("[Index actualizer] [triggerID: %s] is nil, remove from index", triggerID)
			index.index.Delete(triggerID)
		} else {
			triggersToUpdate = append(triggersToUpdate, trigger.ID)
			index.logger.Debugf("[Index actualizer] [triggerID: %s] need to be reindexed...", triggerID)
		}
	}
	if len(triggersToUpdate) > 0 {
		count, err2 := index.addTriggers(triggersToUpdate, defaultIndexBatchSize)
		index.logger.Debugf("[Index actualizer] %d triggers reindexed", count)
		if err2 != nil {
			return err2
		}
	}
	return nil
}
