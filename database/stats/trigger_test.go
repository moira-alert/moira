package stats

import (
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_metrics "github.com/moira-alert/moira/mock/moira-alert/metrics"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestTriggerStatsCheckTriggerCount(t *testing.T) {
	Convey("Given db returns correct results", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		registry := mock_metrics.NewMockRegistry(mockCtrl)

		graphiteLocalCount := int64(12)
		graphiteRemoteCount := int64(24)
		prometheusRemoteCount := int64(42)

		graphiteLocalMeter := mock_metrics.NewMockMeter(mockCtrl)
		graphiteRemoteMeter := mock_metrics.NewMockMeter(mockCtrl)
		prometheusRemoteMeter := mock_metrics.NewMockMeter(mockCtrl)

		registry.EXPECT().NewMeter("triggers", moira.GraphiteLocal.String(), moira.DefaultCluster.String()).Return(graphiteLocalMeter)
		registry.EXPECT().NewMeter("triggers", moira.GraphiteRemote.String(), moira.DefaultCluster.String()).Return(graphiteRemoteMeter)
		registry.EXPECT().NewMeter("triggers", moira.PrometheusRemote.String(), moira.DefaultCluster.String()).Return(prometheusRemoteMeter)

		database := mock_moira_alert.NewMockDatabase(mockCtrl)
		database.EXPECT().GetTriggerCount(gomock.Any()).Return(map[moira.ClusterKey]int64{
			moira.DefaultLocalCluster:            graphiteLocalCount,
			moira.DefaultGraphiteRemoteCluster:   graphiteRemoteCount,
			moira.DefaultPrometheusRemoteCluster: prometheusRemoteCount,
		}, nil)

		graphiteLocalMeter.EXPECT().Mark(graphiteLocalCount)
		graphiteRemoteMeter.EXPECT().Mark(graphiteRemoteCount)
		prometheusRemoteMeter.EXPECT().Mark(prometheusRemoteCount)

		logger, _ := zerolog_adapter.GetLogger("Test")
		clusters := []moira.ClusterKey{
			moira.DefaultLocalCluster,
			moira.DefaultGraphiteRemoteCluster,
			moira.DefaultPrometheusRemoteCluster,
		}
		triggerStats := NewTriggerStats(registry, database, logger, clusters)

		triggerStats.checkTriggerCount()
	})
}
