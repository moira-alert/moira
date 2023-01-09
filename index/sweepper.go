package index

import "time"

const (
	sweeperTimeToKeep  = time.Hour
	sweeperRunInterval = time.Minute
)

func (index *Index) runTriggersToReindexSweepper() error {
	ticker := time.NewTicker(sweeperRunInterval)
	index.logger.Infob().
		String("trigger_time_to_keep", sweeperTimeToKeep.String()).
		String("time_between_sweeps", sweeperRunInterval.String()).
		Msg("Start triggers to reindex sweepper: remove outdated triggers from redis")

	for {
		select {
		case <-index.tomb.Dying():
			index.logger.Infob().Msg("Stop index sweepper")
			return nil
		case <-ticker.C:
			timeToDelete := time.Now().Add(-sweeperTimeToKeep).Unix()
			if err := index.database.RemoveTriggersToReindex(timeToDelete); err != nil {
				index.logger.Warningb().
					Error(err).
					Msg("Cannot sweep triggers to reindex from redis")
			}
		}
	}
}
