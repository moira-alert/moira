package metrics

import "github.com/moira-alert/moira"

const triggersMetricsPrefix = "triggersMetrics"

type TriggersMetrics struct {
	countByTriggerSource map[moira.TriggerSource]Histogram
}

func ConfigureTriggersMetrics(registry Registry) *TriggersMetrics {
	return &TriggersMetrics{
		countByTriggerSource: map[moira.TriggerSource]Histogram{
			moira.GraphiteLocal:    registry.NewHistogram(triggersMetricsPrefix, string(moira.GraphiteLocal)),
			moira.GraphiteRemote:   registry.NewHistogram(triggersMetricsPrefix, string(moira.GraphiteRemote)),
			moira.PrometheusRemote: registry.NewHistogram(triggersMetricsPrefix, string(moira.PrometheusRemote)),
		},
	}
}

func (metrics *TriggersMetrics) Update(source moira.TriggerSource, count int64) {
	metrics.countByTriggerSource[source].Update(count)
}
