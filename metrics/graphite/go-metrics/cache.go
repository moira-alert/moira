package metrics

import (
	"github.com/moira-alert/moira-alert/metrics/graphite"
)

// ConfigureCacheMetrics initialize graphite metrics
func ConfigureCacheMetrics() *graphite.CacheMetrics {
	return &graphite.CacheMetrics{
		TotalMetricsReceived:    newRegisteredMeter("received.total"),
		ValidMetricsReceived:    newRegisteredMeter("received.valid"),
		MatchingMetricsReceived: newRegisteredMeter("received.matching"),
		MatchingTimer:           newRegisteredTimer("time.match"),
		SavingTimer:             newRegisteredTimer("time.save"),
		BuildTreeTimer:          newRegisteredTimer("time.buildtree"),
		TotalReceived:           0,
		ValidReceived:           0,
		MatchedReceived:         0,
	}
}
