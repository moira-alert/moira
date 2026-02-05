package metrics

import (
	"github.com/moira-alert/moira"
)

// TriggersMetrics Collection of metrics for trigger count metrics.
type TriggersMetrics struct {
	triggerCounts map[moira.ClusterKey]Meter
}

// NewTriggersMetrics Creates and configurates the instance of TriggersMetrics.
func NewTriggersMetrics(registry Registry, attributedRegistry MetricRegistry, clusterKeys []moira.ClusterKey) (*TriggersMetrics, error) {
	meters := make(map[moira.ClusterKey]Meter, len(clusterKeys))

	for _, key := range clusterKeys {
		attributedReg := attributedRegistry.WithAttributes(Attributes{
			Attribute{Key: "trigger_source", Value: key.TriggerSource.String()},
			Attribute{Key: "cluster_id", Value: key.ClusterId.String()},
		})

		attributedMeter, err := attributedReg.NewGauge("triggers_count")
		if err != nil {
			return nil, err
		}

		meters[key] = NewCompositeMeter(registry.NewMeter("triggers", key.TriggerSource.String(), key.ClusterId.String()), attributedMeter)
	}

	return &TriggersMetrics{
		triggerCounts: meters,
	}, nil
}

// Mark Marks the number of trigger for given trigger source.
func (metrics *TriggersMetrics) Mark(source moira.ClusterKey, count int64) {
	metrics.triggerCounts[source].Mark(count)
}
