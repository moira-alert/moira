package worker

import (
	"time"

	"github.com/moira-alert/moira/remote"
)

func (worker *Checker) remoteChecker() error {
	checkTicker := time.NewTicker(worker.RemoteConfig.CheckInterval)
	for {
		select {
		case <-worker.tomb.Dying():
			checkTicker.Stop()
			worker.Logger.Info("Remote checker stopped")
			return nil
		case <-checkTicker.C:
			if err := worker.checkRemote(); err != nil {
				worker.Logger.Errorf("Remote checker failed: %s", err.Error())
			}
		}
	}
}

func (worker *Checker) checkRemote() error {
	remoteAvailable, err := remote.IsRemoteAvailable(worker.RemoteConfig)
	if !remoteAvailable {
		worker.Logger.Infof("Remote API is unavailable. Stop checking remote triggers. Error: %s", err.Error())
	} else {
		worker.Logger.Debug("Checking remote triggers")
		triggerIds, err := worker.Database.GetRemoteTriggerIDs()
		if err != nil {
			return err
		}
		worker.addRemoteTriggerIDsIfNeeded(triggerIds)
	}
	return nil
}
