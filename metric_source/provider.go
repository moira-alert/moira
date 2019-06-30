package metricSource

import (
	"fmt"

	"github.com/moira-alert/moira"
)

// ErrMetricSourceIsNotConfigured is used then metric source return false on IsConfigured method call with nil error
var ErrMetricSourceIsNotConfigured = fmt.Errorf("metric source is not configured")

// SourceProvider is a provider for all known metrics sources
type SourceProvider struct {
	local      MetricSource
	graphite   MetricSource
	prometheus MetricSource
}

// CreateMetricSourceProvider just creates SourceProvider with all known metrics sources
func CreateMetricSourceProvider(local MetricSource, graphite MetricSource, prometheus MetricSource) *SourceProvider {
	return &SourceProvider{
		prometheus: prometheus,
		graphite:   graphite,
		local:      local,
	}
}

// GetLocal gets local metric source. If it not configured returns not empty error
func (provider *SourceProvider) GetLocal() (MetricSource, error) {
	return returnSource(provider.local)
}

// GetGraphite gets graphite metric source. If it not configured returns not empty error
func (provider *SourceProvider) GetGraphite() (MetricSource, error) {
	return returnSource(provider.graphite)
}

// GetPrometheus gets prometheus metric source. If it not configured returns not empty error
func (provider *SourceProvider) GetPrometheus() (MetricSource, error) {
	return returnSource(provider.prometheus)
}

// GetTriggerMetricSource get metrics source by given trigger. If it not configured returns not empty error
func (provider *SourceProvider) GetTriggerMetricSource(trigger *moira.Trigger) (MetricSource, error) {
	return provider.GetMetricSource(trigger.SourceType)
}

// GetMetricSource return metric source depending on trigger flag: is graphite trigger or not. GetLocal if not.
func (provider *SourceProvider) GetMetricSource(sourceType string) (MetricSource, error) {
	switch sourceType {
	case moira.Graphite:
		return provider.GetGraphite()
	case moira.Prometheus:
		return provider.GetPrometheus()
	default:
		return provider.GetLocal()
	}
}

func returnSource(source MetricSource) (MetricSource, error) {
	isConfigured, err := source.IsConfigured()
	if !isConfigured && err == nil {
		return source, ErrMetricSourceIsNotConfigured
	}
	return source, err
}
