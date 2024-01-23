package worker

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/patrickmn/go-cache"
)

func (manager *WorkerManager) newMetricsHandler(metricEventsChannel <-chan *moira.MetricEvent) error {
	for {
		metricEvent, ok := <-metricEventsChannel
		if !ok {
			return nil
		}
		pattern := metricEvent.Pattern
		if manager.needHandlePattern(pattern) {
			if err := manager.handleMetricEvent(pattern); err != nil {
				manager.Logger.Error().
					Error(err).
					Msg("Failed to handle metricEvent")
			}
		}
	}
}

func (manager *WorkerManager) needHandlePattern(pattern string) bool {
	err := manager.PatternCache.Add(pattern, true, cache.DefaultExpiration)
	return err == nil
}

func (manager *WorkerManager) handleMetricEvent(pattern string) error {
	start := time.Now()
	defer manager.Metrics.MetricEventsHandleTime.UpdateSince(start)

	manager.lastData = time.Now().UTC().Unix()
	triggerIds, err := manager.Database.GetPatternTriggerIDs(pattern)
	if err != nil {
		return err
	}

	// Cleanup pattern and its metrics if this pattern doesn't match to any trigger
	if len(triggerIds) == 0 {
		if err := manager.Database.RemovePatternWithMetrics(pattern); err != nil {
			return err
		}
	}

	manager.addLocalTriggerIDsIfNeeded(triggerIds)
	return nil
}
