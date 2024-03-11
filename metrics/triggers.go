package metrics

import (
	"github.com/moira-alert/moira"
)

// Collection of metrics for trigger count metrics
type TriggersMetrics struct {
	triggerCounts map[moira.ClusterKey]Meter
}

// Creates and configurates the instance of TriggersMetrics
func NewTriggersMetrics(registry Registry, clusterKeys []moira.ClusterKey) *TriggersMetrics {
	meters := make(map[moira.ClusterKey]Meter, len(clusterKeys))
	for _, key := range clusterKeys {
		meters[key] = registry.NewMeter("triggers", key.TriggerSource.String(), key.ClusterId.String())
	}

	return &TriggersMetrics{
		triggerCounts: meters,
	}
}

// Marks the number of trigger for given trigger source
func (metrics *TriggersMetrics) Mark(source moira.ClusterKey, count int64) {
	metrics.triggerCounts[source].Mark(count)
}
