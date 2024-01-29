package cmd

import (
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/metric_source/prometheus"
	"github.com/moira-alert/moira/metric_source/remote"
)

func InitMetricSources(remotes RemotesConfig, database moira.Database, logger moira.Logger) (*metricSource.SourceProvider, error) {
	provider := metricSource.CreateMetricSourceProvider()
	provider.RegisterSource(moira.DefaultLocalCluster, local.Create(database))

	for _, graphite := range remotes.Graphite {
		config := graphite.GetRemoteSourceSettings()
		source, err := remote.Create(config)
		if err != nil {
			return nil, err
		}
		provider.RegisterSource(moira.MakeClusterKey(moira.GraphiteRemote, graphite.ClusterId), source)
	}

	for _, prom := range remotes.Prometheus {
		config := prom.GetPrometheusSourceSettings()
		source, err := prometheus.Create(config, logger)
		if err != nil {
			return nil, err
		}
		provider.RegisterSource(moira.MakeClusterKey(moira.PrometheusRemote, prom.ClusterId), source)
	}

	return provider, nil
}
