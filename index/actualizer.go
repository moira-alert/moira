package index

import (
	"time"

	"github.com/moira-alert/moira"
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
				index.logger.Errorf("Index was actualized too far ago. Index actualized: %s. Current time: %s. Should actualize every: %v. Maximum possible interval without actualization: %s. Restart moira-API service to solve this issue",
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

	log := index.logger.Clone().String(moira.LogFieldNameContext, "Index actualizer")
	log.Debugf("Got %d triggers to actualize", len(triggerToReindexIDs))

	triggersToReindex, err := index.database.GetTriggerChecks(triggerToReindexIDs)
	if err != nil {
		return err
	}
	triggersToUpdate := make([]*moira.TriggerCheck, 0)
	triggersToDelete := make([]string, 0)

	for i, triggerID := range triggerToReindexIDs {
		trigger := triggersToReindex[i]

		triggerLog := log.Clone().String(moira.LogFieldNameTriggerID, triggerID)
		if trigger == nil {
			triggersToDelete = append(triggersToDelete, triggerID)
			triggerLog.Debug("Trigger is nil, remove from index")
		} else {
			triggersToUpdate = append(triggersToUpdate, trigger)
			triggerLog.Debug("Trigger need to be reindexed...")
		}
	}

	if len(triggersToDelete) > 0 {
		err2 := index.triggerIndex.Delete(triggersToDelete)
		if err2 != nil {
			return err2
		}
		log.Debugf("%d triggers deleted", len(triggersToDelete))
	}

	if len(triggersToUpdate) > 0 {
		err2 := index.triggerIndex.Write(triggersToUpdate)
		if err2 != nil {
			return err2
		}
		log.Debugf("%d triggers reindexed", len(triggersToUpdate))
	}
	return nil
}
