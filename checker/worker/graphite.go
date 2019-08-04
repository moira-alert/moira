package worker

import (
	"time"

	"github.com/moira-alert/moira/metric_source/remote"
)

func (worker *Checker) graphiteChecker() error {
	checkTicker := time.NewTicker(worker.GraphiteConfig.CheckInterval)
	for {
		select {
		case <-worker.tomb.Dying():
			checkTicker.Stop()
			worker.Logger.Info("Graphite checker stopped")
			return nil
		case <-checkTicker.C:
			if err := worker.checkGraphite(); err != nil {
				worker.Logger.Errorf("Graphite checker failed: %s", err.Error())
			}
		}
	}
}

func (worker *Checker) checkGraphite() error {
	source, err := worker.SourceProvider.GetGraphite()
	if err != nil {
		return err
	}
	remoteAvailable, err := source.(*remote.Graphite).IsRemoteAvailable()
	if !remoteAvailable {
		worker.Logger.Infof("Graphite API is unavailable. Stop checking graphite triggers. Error: %s", err.Error())
	} else {
		worker.Logger.Debug("Checking graphite triggers")
		triggerIds, err := worker.Database.GetGraphiteTriggerIDs()
		if err != nil {
			return err
		}
		worker.addGraphiteTriggerIDsIfNeeded(triggerIds)
	}
	return nil
}
