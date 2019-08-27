package worker

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/patrickmn/go-cache"
)

func (worker *Checker) newMetricsHandler(metricEventsChannel <-chan *moira.MetricEvent) error {
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
	needToCheckTriggerIDs := worker.getTriggerIDsToCheck(triggerIDs)
	if len(needToCheckTriggerIDs) > 0 {
		worker.Database.AddLocalTriggersToCheck(needToCheckTriggerIDs)
	}
}

func (worker *Checker) addGraphiteTriggerIDsIfNeeded(triggerIDs []string) {
	needToCheckGraphiteTriggerIDs := worker.getTriggerIDsToCheck(triggerIDs)
	if len(needToCheckGraphiteTriggerIDs) > 0 {
		worker.Database.AddGraphiteTriggersToCheck(needToCheckGraphiteTriggerIDs)
	}
}

func (worker *Checker) addPrometheusTriggerIDsIfNeeded(triggerIDs []string) {
	needToCheckPrometheusTriggerIDs := worker.getTriggerIDsToCheck(triggerIDs)
	if len(needToCheckPrometheusTriggerIDs) > 0 {
		worker.Database.AddPrometheusTriggersToCheck(needToCheckPrometheusTriggerIDs)
	}
}

func (worker *Checker) getTriggerIDsToCheck(triggerIDs []string) []string {
	lazyTriggerIDs := worker.lazyTriggerIDs.Load().(map[string]bool)
	triggerIDsToCheck := make([]string, len(triggerIDs))
	for _, triggerID := range triggerIDs {
		if _, ok := lazyTriggerIDs[triggerID]; ok {
			randomDuration := worker.getRandomLazyCacheDuration()
			if err := worker.LazyTriggersCache.Add(triggerID, true, randomDuration); err != nil {
				continue
			}
		}
		//if err := worker.TriggerCache.Add(triggerID, true, cache.DefaultExpiration); err == nil {
		triggerIDsToCheck = append(triggerIDsToCheck, triggerID)
		//}
	}
	return triggerIDsToCheck
}
