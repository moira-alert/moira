package index

import (
	"time"

	"github.com/moira-alert/moira"
)

const actualizerRunInterval = time.Second

func (index *Index) runIndexActualizer() error {
	ticker := time.NewTicker(actualizerRunInterval)
	index.logger.Info().
		Interface("actualizer_interval", actualizerRunInterval).
		Msg("Start index actualizer: reindex changed triggers in loop with given interval")

	for {
		select {
		case <-index.tomb.Dying():
			index.logger.Info().Msg("Stop index actualizer")
			return nil
		case <-ticker.C:
			newTime := time.Now().Unix()
			if float64(newTime-index.indexActualizedTS) > sweeperTimeToKeep.Seconds() {
				index.logger.Error().
					String("index_actualized_at", time.Unix(index.indexActualizedTS, 0).Format(time.RFC3339)).
					String("current_time", time.Now().Format(time.RFC3339)).
					String("actualization_interval", actualizerRunInterval.String()).
					String("max_interval_without_actualization", sweeperTimeToKeep.String()).
					Msg("Index was actualized too far ago. Restart moira-API service to solve this issue")
			}
			if err := index.actualizeIndex(); err != nil {
				index.logger.Warning().
					Error(err).
					Msg("Cannot actualize triggers")
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
	log.Debug().
		Int("triggers_count", len(triggerToReindexIDs)).
		Msg("Got triggers to actualize")

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
			triggerLog.Debug().Msg("Trigger is nil, remove from index")
		} else {
			triggersToUpdate = append(triggersToUpdate, trigger)
			triggerLog.Debug().Msg("Trigger need to be reindexed...")
		}
	}

	if len(triggersToDelete) > 0 {
		err2 := index.triggerIndex.Delete(triggersToDelete)
		if err2 != nil {
			return err2
		}
		log.Debug().
			Int("triggers_count", len(triggersToDelete)).
			Msg("Some triggers deleted")
	}

	if len(triggersToUpdate) > 0 {
		err2 := index.triggerIndex.Write(triggersToUpdate)
		if err2 != nil {
			return err2
		}
		log.Debug().
			Int("triggers_count", len(triggersToUpdate)).
			Msg("Some triggers reindexed")
	}
	return nil
}
