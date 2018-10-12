package worker

import (
	"time"
)

func (worker *Checker) noDataChecker(stop chan bool) error {
	checkTicker := time.NewTicker(worker.Config.NoDataCheckInterval)
	worker.Logger.Info("NODATA checker started")
	for {
		select {
		case <-stop:
			checkTicker.Stop()
			worker.Logger.Info("NODATA checker stopped")
			return nil
		case <-checkTicker.C:
			if err := worker.checkNoData(); err != nil {
				worker.Logger.Errorf("NODATA check failed: %s", err.Error())
			}
		}
	}
}

func (worker *Checker) checkNoData() error {
	now := time.Now().UTC().Unix()
	if worker.lastData+worker.Config.StopCheckingIntervalSeconds < now {
		worker.Logger.Infof("Checking NODATA disabled. No metrics for %v seconds", now-worker.lastData)
	} else {
		worker.Logger.Info("Checking NODATA")
		triggerIds, err := worker.Database.GetTriggerIDs()
		if err != nil {
			return err
		}
		worker.addTriggerIDsIfNeeded(triggerIds)
	}
	return nil
}

// runNodataChecker starts NODATA checker and manages its subscription in Redis
// to make sure there is always only one working checker
func (worker *Checker) runNodataChecker() error {
	databaseMutexExpiry := worker.Config.NoDataCheckInterval
	singleCheckerStateExpiry := time.Minute
	stop := make(chan bool)

	firstCheck := true
	go func() {
		for {
			if worker.Database.RegisterNodataCheckerIfAlreadyNot(databaseMutexExpiry) {
				worker.Logger.Infof("Registered new NODATA checker, start checking triggers for NODATA")
				go worker.noDataChecker(stop)
				worker.renewRegistration(databaseMutexExpiry, stop)
				continue
			}
			if firstCheck {
				worker.Logger.Infof("NODATA checker already registered, trying for register every %v in loop", singleCheckerStateExpiry)
				firstCheck = false
			}
			<-time.After(singleCheckerStateExpiry)
		}
	}()
	return nil
}

// renewRegistration tries to renew NODATA-checker subscription
// and gracefully stops NODATA checker on fail to prevent multiple checkers running
func (worker *Checker) renewRegistration(ttl time.Duration, stop chan bool) {
	renewTicker := time.NewTicker(ttl / 2)
	for {
		select {
		case <-renewTicker.C:
			if !worker.Database.RenewNodataCheckerRegistration() {
				worker.Logger.Warningf("Could not renew registration for NODATA checker")
				stop <- true
				return
			}
		case <-worker.tomb.Dying():
			renewTicker.Stop()
			stop <- true
			return
		}
	}
}
