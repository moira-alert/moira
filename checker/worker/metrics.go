package worker

import (
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

	check.addLocalTriggerIDsIfNeeded(triggerIds)
	return nil
}
