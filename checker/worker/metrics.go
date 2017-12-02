package worker

import (
	"time"

	"github.com/moira-alert/moira"
)

func (worker *Checker) metricsChecker(metricEventsChannel <-chan *moira.MetricEvent) error {
	for {
		metricEvent, ok := <-metricEventsChannel
		if !ok {
			close(worker.triggersToCheck)
			worker.Logger.Info("Checking for new events stopped")
			return nil
		}
		if err := worker.handleMetricEvent(metricEvent); err != nil {
			worker.Logger.Errorf("Failed to handle metricEvent: %s", err.Error())
		}
	}
}

func (worker *Checker) handleMetricEvent(metricEvent *moira.MetricEvent) error {
	worker.lastData = time.Now().UTC().Unix()
	pattern := metricEvent.Pattern
	metric := metricEvent.Metric

	if err := worker.Database.AddPatternMetric(pattern, metric); err != nil {
		return err
	}
	triggerIds, err := worker.Database.GetPatternTriggerIDs(pattern)
	if err != nil {
		return err
	}
	// Cleanup pattern and its metrics if this pattern doesn't match to any trigger
	if len(triggerIds) == 0 {
		if err := worker.Database.RemovePatternWithMetrics(pattern); err != nil {
			return err
		}
	}
	worker.perform(triggerIds, worker.Config.CheckInterval)
	return nil
}
