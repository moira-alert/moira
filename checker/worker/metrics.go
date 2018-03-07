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
		pattern := metricEvent.Pattern
		if worker.needHandlePattern(pattern, worker.Config.CheckInterval) {
			if err := worker.handleMetricEvent(pattern); err != nil {
				worker.Logger.Errorf("Failed to handle metricEvent: %s", err.Error())
			}
		}
	}
}

func (worker *Checker) needHandlePattern(pattern string, cacheTTL time.Duration) bool {
	err := worker.PatternCache.Add(pattern, true, cacheTTL)
	return err == nil
}

func (worker *Checker) handleMetricEvent(pattern string) error {
	start := time.Now()
	defer worker.Metrics.MetricEventsHandleTime.UpdateSince(start)
	worker.lastData = time.Now().UTC().Unix()
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
	worker.addTriggerIDsIfNeeded(triggerIds, worker.Config.CheckInterval)
	return nil
}

func (worker *Checker) addTriggerIDsIfNeeded(triggerIDs []string, cacheTTL time.Duration) {
	for _, triggerID := range triggerIDs {
		if worker.needHandleTrigger(triggerID, cacheTTL) {
			worker.triggersToCheck <- triggerID
		}
	}
}

func (worker *Checker) needHandleTrigger(triggerID string, cacheTTL time.Duration) bool {
	err := worker.TriggerCache.Add(triggerID, true, cacheTTL)
	return err == nil
}
