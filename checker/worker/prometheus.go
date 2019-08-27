package worker

import (
	"github.com/gosexy/to"
	"github.com/moira-alert/moira/metric_source/prometheus"
	"time"
)

func (worker *Checker) prometheusChecker() error {
	checkTicker := time.NewTicker(to.Duration("5s"))
	for {
		select {
		case <-worker.tomb.Dying():
			checkTicker.Stop()
			worker.Logger.Info("Prometheus checker stopped")
			return nil
		case <-checkTicker.C:
			if err := worker.checkPrometheus(); err != nil {
				worker.Logger.Errorf("Prometheus checker failed: %s", err.Error())
			}
		}
	}
}

func (worker *Checker) checkPrometheus() error {
	source, err := worker.SourceProvider.GetPrometheus()
	if err != nil {
		return err
	}
	isAvailable, err := source.(*prometheus.Source).IsAvailable()
	if !isAvailable {
		worker.Logger.Infof("Prometheus API is unavailable. Stop checking prometheus triggers. Error: %s", err.Error())
	} else {
		worker.Logger.Debug("Checking prometheus triggers")
		triggerIds, err := worker.Database.GetPrometheusTriggerIDs()
		if err != nil {
			return err
		}
		worker.Logger.Debugf("%d prometheus triggers were found", len(triggerIds))
		worker.addPrometheusTriggerIDsIfNeeded(triggerIds)
	}
	return nil
}
