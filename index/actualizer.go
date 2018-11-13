package index

import (
	"time"
)

const actualizerRunInterval = time.Second

func (index *Index) runIndexActualizer() error {
	ticker := time.NewTicker(actualizerRunInterval)
	actualizationTime := time.Now().Add(-time.Minute * 5).Unix()
	index.logger.Infof("Start index actualizer: reindex changed triggers every %v", actualizerRunInterval)

	for {
		select {
		case <-index.tomb.Dying():
			return nil
		case <-ticker.C:
			newTime := time.Now().Unix()
			if err := index.actualizeIndex(actualizationTime); err != nil {
				index.logger.Warningf("Cannot actualize triggers: %s", err.Error())
				continue
			}
			actualizationTime = newTime
		}
	}
}

func (index *Index) actualizeIndex(lastActualizeTs int64) error {
	triggerToReindexIDs, err := index.database.FetchTriggersToReindex(lastActualizeTs)
	if err != nil {
		return err
	}

	if len(triggerToReindexIDs) == 0 {
		return nil
	}

	index.logger.Debugf("[Index actualizer]: got %d triggers from redis", len(triggerToReindexIDs))

	triggersToReindex, err := index.database.GetTriggerChecks(triggerToReindexIDs)
	if err != nil {
		return err
	}

	for i, triggerID := range triggerToReindexIDs {
		trigger := triggersToReindex[i]
		if trigger == nil {
			index.logger.Debugf("[Index actualizer] [triggerID: %s] is nil, remove from index", triggerID)
			index.index.Delete(triggerID)
		} else {
			index.logger.Debugf("[Index actualizer] [triggerID: %s] reindexing...", triggerID)
			err = index.indexTriggerCheck(trigger)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
