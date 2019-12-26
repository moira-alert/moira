// nolint
package metrics

import (
	"github.com/moira-alert/moira/metrics/graphite"
)

// MetersCollection is realization of metrics map of type GetRegisteredMeter
type MetersCollection struct {
	metrics map[string]Meter
}

// newMetersCollection create empty GetRegisteredMeter map
func newMetersCollection() *MetersCollection {
	return &MetersCollection{make(map[string]Meter)}
}

func (meters *MetersCollection) RegisterMeter(name, path string) {
	meters.metrics[name] = *registerMeter(path)
}

func (meters *MetersCollection) GetRegisteredMeter(name string) (graphite.Meter, bool) {
	value, found := meters.metrics[name]
	return &value, found
}
