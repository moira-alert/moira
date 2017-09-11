package metrics

import (
	"fmt"
	"github.com/moira-alert/moira/metrics/graphite"
)

// ConfigureFilterMetrics initialize graphite metrics
func ConfigureFilterMetrics(prefix string) *graphite.FilterMetrics {
	return &graphite.FilterMetrics{
		TotalMetricsReceived:    newRegisteredMeter(metricNameWithPrefix(prefix, "received.total")),
		ValidMetricsReceived:    newRegisteredMeter(metricNameWithPrefix(prefix, "received.valid")),
		MatchingMetricsReceived: newRegisteredMeter(metricNameWithPrefix(prefix, "received.matching")),
		MatchingTimer:           newRegisteredTimer(metricNameWithPrefix(prefix, "time.match")),
		SavingTimer:             newRegisteredTimer(metricNameWithPrefix(prefix, "time.save")),
		BuildTreeTimer:          newRegisteredTimer(metricNameWithPrefix(prefix, "time.buildtree")),
	}
}

// ConfigureNotifierMetrics is notifier metrics configurator
func ConfigureNotifierMetrics(prefix string) *graphite.NotifierMetrics {
	return &graphite.NotifierMetrics{
		SubsMalformed:          newRegisteredMeter(metricNameWithPrefix(prefix, "subs.malformed")),
		EventsReceived:         newRegisteredMeter(metricNameWithPrefix(prefix, "events.received")),
		EventsMalformed:        newRegisteredMeter(metricNameWithPrefix(prefix, "events.malformed")),
		EventsProcessingFailed: newRegisteredMeter(metricNameWithPrefix(prefix, "events.failed")),
		SendingFailed:          newRegisteredMeter(metricNameWithPrefix(prefix, "sending.failed")),
		SendersOkMetrics:       newMetricsMap(),
		SendersFailedMetrics:   newMetricsMap(),
	}
}

// ConfigureCheckerMetrics is checker metrics configurator
func ConfigureCheckerMetrics(prefix string) *graphite.CheckerMetrics {
	return &graphite.CheckerMetrics{
		CheckError:       newRegisteredMeter(metricNameWithPrefix(prefix, "errors.check")),
		HandleError:      newRegisteredMeter(metricNameWithPrefix(prefix, "errors.handle")),
		TriggerCheckTime: newRegisteredTimer(metricNameWithPrefix(prefix, "triggers")),
	}
}

func metricNameWithPrefix(prefix, metric string) string {
	return fmt.Sprintf("%s.%s", prefix, metric)
}
