package main

import (
	"testing"
	"time"

	mocks "github.com/moira-alert/moira/mock/moira-alert"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCleanupOutdatedMetrics(t *testing.T) {
	conf := getDefault()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	db := mocks.NewMockDatabase(mockCtrl)

	Convey("Test cleanup", t, func() {
		db.EXPECT().CleanupOutdatedMetrics(-168 * time.Hour).Return(nil)
		err := cleanupOutdatedMetrics(conf.Cleanup, db)
		So(err, ShouldBeNil)
	})
}
