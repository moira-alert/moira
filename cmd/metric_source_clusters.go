package cmd

import (
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/metric_source/prometheus"
	"github.com/moira-alert/moira/metric_source/remote"
	"github.com/xiam/to"
)

// InitMetricSources initializes SourceProvider from given remote source configs.
func InitMetricSources(remotes RemotesConfig, database moira.Database, logger moira.Logger) (*metricSource.SourceProvider, error) {
	err := remotes.Validate()
	if err != nil {
		return nil, fmt.Errorf("remotes config validation failed: %w", err)
	}

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

// MakeClustersWebConfig initializes cluster list for api web config
func MakeClustersWebConfig(redisConfig RedisConfig, remotes RemotesConfig) []api.MetricSourceCluster {
	clusters := []api.MetricSourceCluster{{
		TriggerSource: moira.GraphiteLocal,
		ClusterId:     moira.DefaultCluster,
		ClusterName:   "Graphite Local",
		MetricsTTL:    uint64(to.Duration(redisConfig.MetricsTTL).Seconds()),
	}}

	for _, remote := range remotes.Graphite {
		cluster := api.MetricSourceCluster{
			TriggerSource: moira.GraphiteRemote,
			ClusterId:     remote.ClusterId,
			ClusterName:   remote.ClusterName,
			MetricsTTL:    uint64(to.Duration(remote.MetricsTTL).Seconds()),
		}
		clusters = append(clusters, cluster)
	}

	for _, remote := range remotes.Prometheus {
		cluster := api.MetricSourceCluster{
			TriggerSource: moira.PrometheusRemote,
			ClusterId:     remote.ClusterId,
			ClusterName:   remote.ClusterName,
			MetricsTTL:    uint64(to.Duration(remote.MetricsTTL).Seconds()),
		}
		clusters = append(clusters, cluster)
	}

	return clusters
}

// MakeClusterList creates a comprehensive list of metric source clusters
func MakeClusterList(remotes RemotesConfig) moira.ClusterList {
	clusterList := moira.ClusterList{moira.DefaultLocalCluster}

	for _, remote := range remotes.Graphite {
		clusterId := moira.MakeClusterKey(moira.GraphiteRemote, remote.ClusterId)
		clusterList = append(clusterList, clusterId)
	}

	for _, remote := range remotes.Prometheus {
		cluster := moira.MakeClusterKey(moira.PrometheusRemote, remote.ClusterId)
		clusterList = append(clusterList, cluster)
	}

	return clusterList
}
