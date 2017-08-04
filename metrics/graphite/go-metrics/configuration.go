package metrics

import (
	"fmt"
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

//ConfigureNotifierMetrics is notifier metrics configurator
func ConfigureNotifierMetrics() *graphite.NotifierMetrics {
	return &graphite.NotifierMetrics{
		EventsReceived:         newRegisteredMeter("events.received"),
		EventsMalformed:        newRegisteredMeter("events.malformed"),
		EventsProcessingFailed: newRegisteredMeter("events.failed"),
		SendingFailed:          newRegisteredMeter("sending.failed"),
		SendersOkMetrics:       newMetricsMap(),
		SendersFailedMetrics:   newMetricsMap(),
	}
}

//ConfigureDatabaseMetrics is database metrics configurator
func ConfigureDatabaseMetrics() *graphite.DatabaseMetrics {
	return &graphite.DatabaseMetrics{
		SubsMalformed: newRegisteredMeter("subs.malformed"),
	}
}

//ConfigureCheckerMetrics is checker metrics configurator
func ConfigureCheckerMetrics(checkerNumber int) *graphite.CheckerMetrics {
	return &graphite.CheckerMetrics{
		CheckerError:      newRegisteredMeter(fmt.Sprintf("checker.errors.%v", checkerNumber)),
		TriggerCheckTime:  newRegisteredTimer(fmt.Sprintf("checker.triggers.%v", checkerNumber)),
		TriggerCheckGauge: newRegisteredGauge(fmt.Sprintf("checker.triggers.%v.sum", checkerNumber)),
	}
}
