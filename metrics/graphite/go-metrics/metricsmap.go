package go_metrics

import (
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"github.com/rcrowley/go-metrics"
)

type MetricsMap struct {
	metrics map[string]metrics.Meter
}

func (metricsMap *MetricsMap) AddMetric(name, path string) {
	metricsMap.metrics[name] = NewRegisteredMeter(path)
}

func (metricsMap *MetricsMap) GetMetric(name string) (graphite.Meter, bool) {
	value, found := metricsMap.metrics[name]
	return value, found
}
