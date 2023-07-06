package metricsource

import (
	"fmt"

	"github.com/moira-alert/moira"
)

// ErrMetricSourceIsNotConfigured is used then metric source return false on IsConfigured method call with nil error
var ErrMetricSourceIsNotConfigured = fmt.Errorf("metric source is not configured")

// SourceProvider is a provider for all known metrics sources
type SourceProvider struct {
	local      MetricSource
	remote     MetricSource
	prometheus MetricSource
}

// CreateMetricSourceProvider just creates SourceProvider with all known metrics sources
func CreateMetricSourceProvider(local, remote, prometheus MetricSource) *SourceProvider {
	return &SourceProvider{
		remote:     remote,
		local:      local,
		prometheus: prometheus,
	}
}

// GetLocal gets local metric source. If it not configured returns not empty error
func (provider *SourceProvider) GetLocal() (MetricSource, error) {
	return returnSource(provider.local)
}

// GetRemote gets remote metric source. If it not configured returns not empty error
func (provider *SourceProvider) GetRemote() (MetricSource, error) {
	return returnSource(provider.remote)
}

// GetRemote gets remote metric source. If it not configured returns not empty error
func (provider *SourceProvider) GetPrometheus() (MetricSource, error) {
	return returnSource(provider.prometheus)
}

// GetTriggerMetricSource get metrics source by given trigger. If it not configured returns not empty error
func (provider *SourceProvider) GetTriggerMetricSource(trigger *moira.Trigger) (MetricSource, error) {
	return provider.GetMetricSource(trigger.TriggerSource)
}

// GetMetricSource return metric source depending on trigger flag: is remote trigger or not. GetLocal if not.
func (provider *SourceProvider) GetMetricSource(triggerSource moira.TriggerSource) (MetricSource, error) {
	switch triggerSource {
	case moira.GraphiteLocal:
		return provider.GetLocal()

	case moira.GraphiteRemote:
		return provider.GetRemote()

	case moira.PrometheusRemote:
		return provider.GetPrometheus()
	}

	return nil, fmt.Errorf("unknown metric source")
}

func returnSource(source MetricSource) (MetricSource, error) {
	isConfigured, err := source.IsConfigured()
	if !isConfigured && err == nil {
		return source, ErrMetricSourceIsNotConfigured
	}
	return source, err
}
