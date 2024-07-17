package main

import (
	"testing"
	"time"

	"github.com/moira-alert/moira/database/redis"
	mocks "github.com/moira-alert/moira/mock/moira-alert"

	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestCleanUpOutdatedMetrics(t *testing.T) {
	conf := getDefault()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	db := mocks.NewMockDatabase(mockCtrl)

	Convey("Test cleanup outdated metrics", t, func() {
		Convey("With valid duration", func() {
			db.EXPECT().CleanUpOutdatedMetrics(-168 * time.Hour).Return(nil)
			err := handleCleanUpOutdatedMetrics(conf.Cleanup, db)
			So(err, ShouldBeNil)
		})

		Convey("With invalid duration", func() {
			conf.Cleanup.CleanupMetricsDuration = "168h"
			db.EXPECT().CleanUpOutdatedMetrics(168 * time.Hour).Return(redis.ErrCleanUpDurationGreaterThanZero)
			err := handleCleanUpOutdatedMetrics(conf.Cleanup, db)
			So(err, ShouldEqual, redis.ErrCleanUpDurationGreaterThanZero)
		})
	})
}

func TestCleanUpFutureMetrics(t *testing.T) {
	conf := getDefault()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	db := mocks.NewMockDatabase(mockCtrl)

	Convey("Test cleanup future metrics", t, func() {
		Convey("With valid duration", func() {
			db.EXPECT().CleanUpFutureMetrics(60 * time.Minute).Return(nil)
			err := handleCleanUpFutureMetrics(conf.Cleanup, db)
			So(err, ShouldBeNil)
		})

		Convey("With invalid duration", func() {
			conf.Cleanup.CleanupFutureMetricsDuration = "-60m"
			db.EXPECT().CleanUpFutureMetrics(-60 * time.Minute).Return(redis.ErrCleanUpDurationLessThanZero)
			err := handleCleanUpFutureMetrics(conf.Cleanup, db)
			So(err, ShouldEqual, redis.ErrCleanUpDurationLessThanZero)
		})
	})
}
