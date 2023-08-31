package metrics

import "github.com/moira-alert/moira"

// Collection of metrics for trigger count metrics
type TriggersMetrics struct {
	countByTriggerSource map[moira.TriggerSource]Meter
}

// Creates and configurates the instance of TriggersMetrics
func NewTriggersMetrics(registry Registry) *TriggersMetrics {
	return &TriggersMetrics{
		countByTriggerSource: map[moira.TriggerSource]Meter{
			moira.GraphiteLocal:    registry.NewMeter("triggers", "count", "source", string(moira.GraphiteLocal)),
			moira.GraphiteRemote:   registry.NewMeter("triggers", "count", "source", string(moira.GraphiteRemote)),
			moira.PrometheusRemote: registry.NewMeter("triggers", "count", "source", string(moira.PrometheusRemote)),
		},
	}
}

// Marks the number of triger for given trigger source
func (metrics *TriggersMetrics) Mark(source moira.TriggerSource, count int64) {
	metrics.countByTriggerSource[source].Mark(count)
}
