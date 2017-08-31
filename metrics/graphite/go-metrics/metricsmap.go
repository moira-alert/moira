// nolint
package metrics

import (
	"github.com/moira-alert/moira-alert/metrics/graphite"
)

// MetricMap is realization of MetricsMap
type MetricMap struct {
	metrics map[string]Meter
}

func newMetricsMap() *MetricMap {
	return &MetricMap{make(map[string]Meter)}
}

func (metricsMap *MetricMap) AddMetric(name, path string) {
	metricsMap.metrics[name] = *newRegisteredMeter(path)
}

func (metricsMap *MetricMap) GetMetric(name string) (graphite.Meter, bool) {
	value, found := metricsMap.metrics[name]
	return &value, found
}
