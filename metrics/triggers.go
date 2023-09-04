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
			moira.GraphiteLocal:    registry.NewMeter("triggers", string(moira.GraphiteLocal), "count"),
			moira.GraphiteRemote:   registry.NewMeter("triggers", string(moira.GraphiteRemote), "count"),
			moira.PrometheusRemote: registry.NewMeter("triggers", string(moira.PrometheusRemote), "count"),
		},
	}
}

// Marks the number of trigger for given trigger source
func (metrics *TriggersMetrics) Mark(source moira.TriggerSource, count int64) {
	metrics.countByTriggerSource[source].Mark(count)
}
