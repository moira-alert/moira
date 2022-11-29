package index

import "time"

const (
	sweeperTimeToKeep  = time.Hour
	sweeperRunInterval = time.Minute
)

func (index *Index) runTriggersToReindexSweepper() error {
	ticker := time.NewTicker(sweeperRunInterval)
	index.logger.Infof("Start triggers to reindex sweepper: remove outdated (> %v) triggers from redis every %v", sweeperTimeToKeep, sweeperRunInterval)

	for {
		select {
		case <-index.tomb.Dying():
			index.logger.Info("Stop index sweepper")
			return nil
		case <-ticker.C:
			timeToDelete := time.Now().Add(-sweeperTimeToKeep).Unix()
			if err := index.database.RemoveTriggersToReindex(timeToDelete); err != nil {
				index.logger.WarningWithError("Cannot sweep triggers to reindex from redis", err)
			}
		}
	}
}
