package main

import (
	"testing"
	"time"

	mocks "github.com/moira-alert/moira/mock/moira-alert"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCleanUpOutdatedMetrics(t *testing.T) {
	conf := getDefault()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	db := mocks.NewMockDatabase(mockCtrl)

	Convey("Test cleanup", t, func() {
		db.EXPECT().CleanUpOutdatedMetrics(-168 * time.Hour).Return(nil)
		err := handleCleanUpOutdatedMetrics(conf.Cleanup, db)
		So(err, ShouldBeNil)
	})
}
