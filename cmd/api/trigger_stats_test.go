package main

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_metrics "github.com/moira-alert/moira/mock/moira-alert/metrics"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTriggerStatsCheckTriggerCount(t *testing.T) {
	Convey("Given db returns correct results", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		registry := mock_metrics.NewMockRegistry(mockCtrl)

		graphiteLocalCount := int64(12)
		graphiteRemoteCount := int64(24)
		promethteusRemoteCount := int64(42)

		graphiteLocalMeter := mock_metrics.NewMockMeter(mockCtrl)
		graphiteRemoteMeter := mock_metrics.NewMockMeter(mockCtrl)
		promethteusRemoteMeter := mock_metrics.NewMockMeter(mockCtrl)

		registry.EXPECT().NewMeter("triggers", string(moira.GraphiteLocal), "count").Return(graphiteLocalMeter)
		registry.EXPECT().NewMeter("triggers", string(moira.GraphiteRemote), "count").Return(graphiteRemoteMeter)
		registry.EXPECT().NewMeter("triggers", string(moira.PrometheusRemote), "count").Return(promethteusRemoteMeter)

		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
		dataBase.EXPECT().GetTriggerCount().Return(map[moira.TriggerSource]int64{
			moira.GraphiteLocal:    graphiteLocalCount,
			moira.GraphiteRemote:   graphiteRemoteCount,
			moira.PrometheusRemote: promethteusRemoteCount,
		}, nil)

		graphiteLocalMeter.EXPECT().Mark(graphiteLocalCount)
		graphiteRemoteMeter.EXPECT().Mark(graphiteRemoteCount)
		promethteusRemoteMeter.EXPECT().Mark(promethteusRemoteCount)

		logger, _ := zerolog_adapter.GetLogger("Test")
		triggerStats := newTriggerStats(logger, dataBase, registry)

		triggerStats.checkTriggerCount()
	})
}
