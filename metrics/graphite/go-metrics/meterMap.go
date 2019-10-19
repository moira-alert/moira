package metrics

import (
	"github.com/moira-alert/moira/metrics/graphite"
)

// MeterMap is realization of metrics map of type Meter
type MeterMap struct {
	metrics map[string]Meter
}

// newMeterMap create empty Meter map
func newMeterMap() *MeterMap {
	return &MeterMap{make(map[string]Meter)}
}

func (metricsMap *MeterMap) AddMetric(name, path string) {
	metricsMap.metrics[name] = *registerMeter(path)
}

func (metricsMap *MeterMap) GetMetric(name string) (graphite.Meter, bool) {
	value, found := metricsMap.metrics[name]
	return &value, found
}
