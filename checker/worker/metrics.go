package worker

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/patrickmn/go-cache"
)

func (worker *Checker) metricsChecker(metricEventsChannel <-chan *moira.MetricEvent) error {
	for {
		metricEvent, ok := <-metricEventsChannel
		if !ok {
			return nil
		}
		pattern := metricEvent.Pattern
		if worker.needHandlePattern(pattern) {
			if err := worker.handleMetricEvent(pattern); err != nil {
				worker.Logger.Errorf("Failed to handle metricEvent: %s", err.Error())
			}
		}
	}
}

func (worker *Checker) needHandlePattern(pattern string) bool {
	err := worker.PatternCache.Add(pattern, true, cache.DefaultExpiration)
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
	worker.addTriggerIDsIfNeeded(triggerIds)
	return nil
}

func (worker *Checker) addTriggerIDsIfNeeded(triggerIDs []string) {
	needToCheckTriggerIDs := make([]string, len(triggerIDs))
	for _, triggerID := range triggerIDs {
		if worker.needHandleTrigger(triggerID) {
			needToCheckTriggerIDs = append(needToCheckTriggerIDs, triggerID)
		}
	}
	if len(needToCheckTriggerIDs) > 0 {
		worker.Database.AddTriggersToCheck(needToCheckTriggerIDs)
	}
}

func (worker *Checker) addRemoteTriggerIDsIfNeeded(triggerIDs []string) {
	needToCheckRemoteTriggerIDs := make([]string, len(triggerIDs))
	for _, triggerID := range triggerIDs {
		if worker.needHandleTrigger(triggerID) {
			needToCheckRemoteTriggerIDs = append(needToCheckRemoteTriggerIDs, triggerID)
		}
	}
	if len(needToCheckRemoteTriggerIDs) > 0 {
		worker.Database.AddRemoteTriggersToCheck(needToCheckRemoteTriggerIDs)
	}
}

func (worker *Checker) needHandleTrigger(triggerID string) bool {
	if _, ok := worker.lazyTriggerIDs[triggerID]; ok {
		err := worker.LazyTriggersCache.Add(triggerID, true, cache.DefaultExpiration)
		if err != nil {
			return false
		}
	}
	err := worker.TriggerCache.Add(triggerID, true, cache.DefaultExpiration)
	return err == nil
}
