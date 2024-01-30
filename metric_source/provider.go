package metricsource

import (
	"fmt"

	"github.com/moira-alert/moira"
)

// SourceProvider is a provider for all known metrics sources
type SourceProvider struct {
	sources map[moira.ClusterKey]MetricSource
}

// CreateMetricSourceProvider just creates SourceProvider with all known metrics sources
func CreateMetricSourceProvider() *SourceProvider {
	return &SourceProvider{
		sources: make(map[moira.ClusterKey]MetricSource),
	}
}

func CreateTestMetricSourceProvider(local, graphiteRemote, prometheusRemote MetricSource) *SourceProvider {
	provider := CreateMetricSourceProvider()

	if local != nil {
		provider.RegisterSource(moira.DefaultLocalCluster, local)
	}
	if graphiteRemote != nil {
		provider.RegisterSource(moira.DefaultGraphiteRemoteCluster, graphiteRemote)
	}
	if prometheusRemote != nil {
		provider.RegisterSource(moira.MakeClusterKey(moira.PrometheusRemote, moira.DefaultCluster), prometheusRemote)
	}

	return provider
}

func (provider *SourceProvider) RegisterSource(clusterKey moira.ClusterKey, source MetricSource) {
	provider.sources[clusterKey] = source
}

func (provider *SourceProvider) GetAllSources() map[moira.ClusterKey]MetricSource {
	return provider.sources
}

func (provider *SourceProvider) GetClusterList() []moira.ClusterKey {
	result := make([]moira.ClusterKey, 0, len(provider.sources))

	for key := range provider.sources {
		result = append(result, key)
	}

	return result
}

// GetTriggerMetricSource get metrics source by given trigger. If it not configured returns not empty error
func (provider *SourceProvider) GetTriggerMetricSource(trigger *moira.Trigger) (MetricSource, error) {
	return provider.GetMetricSource(trigger.ClusterKey())
}

// GetMetricSource return metric source depending on trigger flag: is remote trigger or not. GetLocal if not.
func (provider *SourceProvider) GetMetricSource(clusterKey moira.ClusterKey) (MetricSource, error) {
	if source, ok := provider.sources[clusterKey]; ok {
		return source, nil
	}

	return nil, fmt.Errorf("unknown metric source with cluster key `%s`", clusterKey.String())
}
