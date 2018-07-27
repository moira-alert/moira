package metrics

import (
	"fmt"

	"github.com/moira-alert/moira/metrics/graphite"
)

// ConfigureFilterMetrics initialize graphite metrics
func ConfigureFilterMetrics(prefix string) *graphite.FilterMetrics {
	return &graphite.FilterMetrics{
		TotalMetricsReceived:    registerCounter(metricNameWithPrefix(prefix, "received.total")),
		ValidMetricsReceived:    registerCounter(metricNameWithPrefix(prefix, "received.valid")),
		MatchingMetricsReceived: registerCounter(metricNameWithPrefix(prefix, "received.matching")),
		MatchingTimer:           registerTimer(metricNameWithPrefix(prefix, "time.match")),
		SavingTimer:             registerTimer(metricNameWithPrefix(prefix, "time.save")),
		BuildTreeTimer:          registerTimer(metricNameWithPrefix(prefix, "time.buildtree")),
		MetricChannelLen:        registerHistogram(metricNameWithPrefix(prefix, "metricsToSave")),
		LineChannelLen:          registerHistogram(metricNameWithPrefix(prefix, "linesToMatch")),
	}
}

// ConfigureNotifierMetrics is notifier metrics configurator
func ConfigureNotifierMetrics(prefix string) *graphite.NotifierMetrics {
	return &graphite.NotifierMetrics{
		SubsMalformed:          registerMeter(metricNameWithPrefix(prefix, "subs.malformed")),
		EventsReceived:         registerMeter(metricNameWithPrefix(prefix, "events.received")),
		EventsMalformed:        registerMeter(metricNameWithPrefix(prefix, "events.malformed")),
		EventsProcessingFailed: registerMeter(metricNameWithPrefix(prefix, "events.failed")),
		SendingFailed:          registerMeter(metricNameWithPrefix(prefix, "sending.failed")),
		SendersOkMetrics:       newMeterMap(),
		SendersFailedMetrics:   newMeterMap(),
	}
}

// ConfigureCheckerMetrics is checker metrics configurator
func ConfigureCheckerMetrics(prefix string) *graphite.CheckerMetrics {
	return &graphite.CheckerMetrics{
		CheckError:                registerMeter(metricNameWithPrefix(prefix, "errors.check")),
		HandleError:               registerMeter(metricNameWithPrefix(prefix, "errors.handle")),
		TriggersCheckTime:         registerTimer(metricNameWithPrefix(prefix, "triggers")),
		TriggerCheckTime:          newTimerMap(metricNameWithPrefix(prefix, "trigger")),
		TriggersToCheckChannelLen: registerHistogram(metricNameWithPrefix(prefix, "triggersToCheck")),
		MetricEventsChannelLen:    registerHistogram(metricNameWithPrefix(prefix, "metricEvents")),
		MetricEventsHandleTime:    registerTimer(metricNameWithPrefix(prefix, "metricEventsHandle")),
	}
}

func metricNameWithPrefix(prefix, metric string) string {
	return fmt.Sprintf("%s.%s", prefix, metric)
}
