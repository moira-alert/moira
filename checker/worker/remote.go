package worker

import "time"

func (worker *Checker) remoteChecker() error {
	checkTicker := time.NewTicker(worker.RemoteConfig.CheckInterval)
	for {
		select {
		case <-worker.tomb.Dying():
			checkTicker.Stop()
			worker.Logger.Info("Remote checker stopped")
			return nil
		case <-checkTicker.C:
			if err := worker.check(); err != nil {
				worker.Logger.Errorf("Remote checker failed: %s", err.Error())
			}
		}
	}
}

func (worker *Checker) check() error {
	worker.Logger.Debug("Checking remote triggers")
	triggerIds, err := worker.Database.GetRemoteTriggerIDs()
	if err != nil {
		return err
	}
	worker.addRemoteTriggerIDsIfNeeded(triggerIds)
	return nil
}
