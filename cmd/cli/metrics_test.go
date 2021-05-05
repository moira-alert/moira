package main

import (
	"errors"
	"github.com/golang/mock/gomock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"github.com/spf13/viper"
	"testing"
	"time"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCleanupOutdatedMetrics(t *testing.T) {
	conf := getDefault()
	logger, err := logging.ConfigureLog(conf.LogFile, conf.LogLevel, "cli", conf.LogPrettyFormat)
	if err != nil {
		t.Fatal(err)
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	db := mock_moira_alert.NewMockDatabase(mockCtrl)
	cursor := mock_moira_alert.NewMockMetricsDatabaseCursor(mockCtrl)

	var duration time.Duration
	Convey("Test prepare for cleanup", t, func() {
		duration, err = time.ParseDuration("-3600s")
		So(err, ShouldBeNil)
		conf.CleanupMetrics.HotParams.CleanupDuration = "-3600s"
		conf.CleanupMetrics.HotParams.CleanupBatchCount = 2
	})
	viper.Set("hot_params", "hot_params:\n  cleanup_duration: \"-3600s\"\n  cleanup_batch: 2\n"+
		"  cleanup_batch_timeout_seconds: 1\n  cleanup_keyscan_batch: 1000")

	Convey("Test simple cleanup", t, func() {
		db.EXPECT().ScanMetricNames().Return(cursor)
		metricsKeys := []string{"testing.metric1"}
		cursor.EXPECT().Next().Return(metricsKeys, nil).Times(1)
		cursor.EXPECT().Next().Return(nil, errors.New("end reached")).Times(1)
		db.EXPECT().RemoveMetricsValues(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		err := cleanupOutdatedMetrics(conf.CleanupMetrics, db, duration, logger)
		So(err, ShouldBeNil)
	})

	Convey("Test batched cleanup", t, func() {
		db.EXPECT().ScanMetricNames().Return(cursor)
		metricsKeys := make([]string, 4)
		cursor.EXPECT().Next().Return(metricsKeys, nil).Times(1)
		cursor.EXPECT().Next().Return(nil, errors.New("end reached")).Times(1)
		db.EXPECT().RemoveMetricsValues(gomock.Any(), gomock.Any()).Return(nil).Times(2)

		err := cleanupOutdatedMetrics(conf.CleanupMetrics, db, duration, logger)
		So(err, ShouldBeNil)
	})
}
