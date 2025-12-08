package stats

import (
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_metrics "github.com/moira-alert/moira/mock/moira-alert/metrics"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTriggerStatsCheckTriggerCount(t *testing.T) {
	const (
		triggersSourceAttribute string = "trigger_source"
		clusterIdAttribute     string = "cluster_id"
	)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	registry := mock_metrics.NewMockRegistry(mockCtrl)
	attributedRegistry := mock_metrics.NewMockMetricRegistry(mockCtrl)

	graphiteLocalCount := int64(12)
	graphiteRemoteCount := int64(24)
	prometheusRemoteCount := int64(42)

	graphiteLocalMeter := mock_metrics.NewMockMeter(mockCtrl)
	graphiteRemoteMeter := mock_metrics.NewMockMeter(mockCtrl)
	prometheusRemoteMeter := mock_metrics.NewMockMeter(mockCtrl)

	registry.EXPECT().NewMeter("triggers", moira.GraphiteLocal.String(), moira.DefaultCluster.String()).Return(graphiteLocalMeter)
	registry.EXPECT().NewMeter("triggers", moira.GraphiteRemote.String(), moira.DefaultCluster.String()).Return(graphiteRemoteMeter)
	registry.EXPECT().NewMeter("triggers", moira.PrometheusRemote.String(), moira.DefaultCluster.String()).Return(prometheusRemoteMeter)
	attributedRegistry.EXPECT().WithAttributes(metrics.Attributes{
		metrics.Attribute{Key: triggersSourceAttribute, Value: moira.GraphiteLocal.String()},
		metrics.Attribute{Key: clusterIdAttribute, Value: moira.DefaultCluster.String()},
	}).Return(attributedRegistry)
	attributedRegistry.EXPECT().WithAttributes(metrics.Attributes{
		metrics.Attribute{Key: triggersSourceAttribute, Value: moira.GraphiteRemote.String()},
		metrics.Attribute{Key: clusterIdAttribute, Value: moira.DefaultCluster.String()},
	}).Return(attributedRegistry)
	attributedRegistry.EXPECT().WithAttributes(metrics.Attributes{
		metrics.Attribute{Key: triggersSourceAttribute, Value: moira.PrometheusRemote.String()},
		metrics.Attribute{Key: clusterIdAttribute, Value: moira.DefaultCluster.String()},
	}).Return(attributedRegistry)
	attributedRegistry.EXPECT().NewGauge("triggers_count").Return(graphiteLocalMeter, nil).Times(1)
	attributedRegistry.EXPECT().NewGauge("triggers_count").Return(graphiteRemoteMeter, nil).Times(1)
	attributedRegistry.EXPECT().NewGauge("triggers_count").Return(prometheusRemoteMeter, nil).Times(1)

	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	database.EXPECT().GetTriggerCount(gomock.Any()).Return(map[moira.ClusterKey]int64{
		moira.DefaultLocalCluster:            graphiteLocalCount,
		moira.DefaultGraphiteRemoteCluster:   graphiteRemoteCount,
		moira.DefaultPrometheusRemoteCluster: prometheusRemoteCount,
	}, nil)

	graphiteLocalMeter.EXPECT().Mark(graphiteLocalCount).Times(2)
	graphiteRemoteMeter.EXPECT().Mark(graphiteRemoteCount).Times(2)
	prometheusRemoteMeter.EXPECT().Mark(prometheusRemoteCount).Times(2)

	logger, _ := zerolog_adapter.GetLogger("Test")
	clusters := []moira.ClusterKey{
		moira.DefaultLocalCluster,
		moira.DefaultGraphiteRemoteCluster,
		moira.DefaultPrometheusRemoteCluster,
	}
	triggerStats, err := NewTriggerStats(registry, attributedRegistry, database, logger, clusters)

	require.NoError(t, err)

	triggerStats.checkTriggerCount()
}
