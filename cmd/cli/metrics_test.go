package main

import (
	"errors"
	"testing"
	"time"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mocks "github.com/moira-alert/moira/mock/moira-alert"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func TestCleanupOutdatedMetrics(t *testing.T) {
	conf := getDefault()
	conf.CleanupMetrics.HotParams.CleanupBatchCount = 2
	conf.CleanupMetrics.HotParams.CleanupBatchTimeoutSeconds = int(time.Second.Seconds())
	conf.CleanupMetrics.HotParams.CleanupDuration = "-3600s"
	conf.CleanupMetrics.HotParams.CleanupKeyScanBatchCount = 1000
	viper.Set("hot_params", conf.CleanupMetrics.HotParams)

	logger, err := logging.ConfigureLog(conf.LogFile, conf.LogLevel, "cli", conf.LogPrettyFormat)
	if err != nil {
		t.Fatal(err)
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	db := mocks.NewMockDatabase(mockCtrl)
	cursor := mocks.NewMockMetricsDatabaseCursor(mockCtrl)

	Convey("Test simple cleanup", t, func() {
		db.EXPECT().ScanMetricNames().Return(cursor)
		metricsKeys := []string{"testing.metric1"}
		cursor.EXPECT().Next().Return(metricsKeys, nil).Times(1)
		cursor.EXPECT().Next().Return(nil, errors.New("end reached")).Times(1)
		db.EXPECT().RemoveMetricsValues(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		err := cleanupOutdatedMetrics(conf.CleanupMetrics, db, logger)
		So(err, ShouldBeNil)
	})

	Convey("Test batched cleanup", t, func() {
		db.EXPECT().ScanMetricNames().Return(cursor)
		metricsKeys := make([]string, 4)
		cursor.EXPECT().Next().Return(metricsKeys, nil).Times(1)
		cursor.EXPECT().Next().Return(nil, errors.New("end reached")).Times(1)
		db.EXPECT().RemoveMetricsValues(gomock.Any(), gomock.Any()).Return(nil).Times(2)

		err := cleanupOutdatedMetrics(conf.CleanupMetrics, db, logger)
		So(err, ShouldBeNil)
	})
}
