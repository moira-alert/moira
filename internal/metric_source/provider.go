package metricSource

import (
	"fmt"

	moira2 "github.com/moira-alert/moira/internal/moira"
)

// ErrMetricSourceIsNotConfigured is used then metric source return false on IsConfigured method call with nil error
var ErrMetricSourceIsNotConfigured = fmt.Errorf("metric source is not configured")

// SourceProvider is a provider for all known metrics sources
type SourceProvider struct {
	local  MetricSource
	remote MetricSource
}

// CreateMetricSourceProvider just creates SourceProvider with all known metrics sources
func CreateMetricSourceProvider(local MetricSource, remote MetricSource) *SourceProvider {
	return &SourceProvider{
		remote: remote,
		local:  local,
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

// GetTriggerMetricSource get metrics source by given trigger. If it not configured returns not empty error
func (provider *SourceProvider) GetTriggerMetricSource(trigger *moira2.Trigger) (MetricSource, error) {
	return provider.GetMetricSource(trigger.IsRemote)
}

// GetMetricSource return metric source depending on trigger flag: is remote trigger or not. GetLocal if not.
func (provider *SourceProvider) GetMetricSource(isRemote bool) (MetricSource, error) {
	if isRemote {
		return provider.GetRemote()
	}
	return provider.GetLocal()
}

func returnSource(source MetricSource) (MetricSource, error) {
	isConfigured, err := source.IsConfigured()
	if !isConfigured && err == nil {
		return source, ErrMetricSourceIsNotConfigured
	}
	return source, err
}
