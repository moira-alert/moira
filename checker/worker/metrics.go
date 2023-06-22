package worker

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/patrickmn/go-cache"
)

func (check *Checker) newMetricsHandler(metricEventsChannel <-chan *moira.MetricEvent) error {
	for {
		metricEvent, ok := <-metricEventsChannel
		if !ok {
			return nil
		}
		pattern := metricEvent.Pattern
		if check.needHandlePattern(pattern) {
			if err := check.handleMetricEvent(pattern); err != nil {
				check.Logger.Error().
					Error(err).
					Msg("Failed to handle metricEvent")
			}
		}
	}
}

func (check *Checker) needHandlePattern(pattern string) bool {
	err := check.PatternCache.Add(pattern, true, cache.DefaultExpiration)
	return err == nil
}

func (check *Checker) handleMetricEvent(pattern string) error {
	start := time.Now()
	defer check.Metrics.MetricEventsHandleTime.UpdateSince(start)
	check.lastData = time.Now().UTC().Unix()
	triggerIds, err := check.Database.GetPatternTriggerIDs(pattern)
	if err != nil {
		return err
	}
	// Cleanup pattern and its metrics if this pattern doesn't match to any trigger
	if len(triggerIds) == 0 {
		if err := check.Database.RemovePatternWithMetrics(pattern); err != nil {
			return err
		}
	}
	check.addTriggerIDsIfNeeded(triggerIds)
	return nil
}

func (check *Checker) addTriggerIDsIfNeeded(triggerIDs []string) {
	needToCheckTriggerIDs := check.getTriggerIDsToCheck(triggerIDs)
	if len(needToCheckTriggerIDs) > 0 {
		check.Database.AddLocalTriggersToCheck(needToCheckTriggerIDs) //nolint
	}
}

func (check *Checker) addRemoteTriggerIDsIfNeeded(triggerIDs []string) {
	needToCheckRemoteTriggerIDs := check.getTriggerIDsToCheck(triggerIDs)
	if len(needToCheckRemoteTriggerIDs) > 0 {
		check.Database.AddRemoteTriggersToCheck(needToCheckRemoteTriggerIDs) //nolint
	}
}

func (check *Checker) addVMSelectTriggerIDsIfNeeded(triggerIDs []string) {
	needToCheckVMSelectTriggerIDs := check.getTriggerIDsToCheck(triggerIDs)
	if len(needToCheckVMSelectTriggerIDs) > 0 {
		check.Logger.Debug().
			String("needToCheckVMSelectTriggerIDs", fmt.Sprintf("%v", needToCheckVMSelectTriggerIDs)).
			Msg("needToCheckVMSelectTriggerIDs")
		check.Database.AddVMSelectTriggersToCheck(needToCheckVMSelectTriggerIDs) //nolint
	}
}

func (check *Checker) getTriggerIDsToCheck(triggerIDs []string) []string {
	lazyTriggerIDs := check.lazyTriggerIDs.Load().(map[string]bool)
	var triggerIDsToCheck []string = make([]string, 0, len(triggerIDs))
	for _, triggerID := range triggerIDs {
		if _, ok := lazyTriggerIDs[triggerID]; ok {
			randomDuration := check.getRandomLazyCacheDuration()
			if err := check.LazyTriggersCache.Add(triggerID, true, randomDuration); err != nil {
				continue
			}
		}
		if err := check.TriggerCache.Add(triggerID, true, cache.DefaultExpiration); err == nil {
			triggerIDsToCheck = append(triggerIDsToCheck, triggerID)
		}
	}
	return triggerIDsToCheck
}
