package redis

import (
	"fmt"
	"testing"

	"github.com/moira-alert/moira/notifier/selfstate"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSelfCheck(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Self state triggers manipulation", t, func() {
		Convey("Empty config", func() {
			count, err := dataBase.GetMetricsUpdatesCount()
			So(count, ShouldEqual, 0)
			So(err, ShouldBeNil)

			count, err = dataBase.GetChecksUpdatesCount()
			So(count, ShouldEqual, 0)
			So(err, ShouldBeNil)

			count, err = dataBase.GetRemoteChecksUpdatesCount()
			So(count, ShouldEqual, 0)
			So(err, ShouldBeNil)
		})

		Convey("Update metrics heartbeat test", func() {
			err := dataBase.UpdateMetricsHeartbeat()
			So(err, ShouldBeNil)

			count, err := dataBase.GetMetricsUpdatesCount()
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})

		Convey("Update metrics checks updates count", func() {
			err := dataBase.SetTriggerLastCheck("123", &lastCheckTest, false)
			So(err, ShouldBeNil)

			count, err := dataBase.GetChecksUpdatesCount()
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)

			err = dataBase.SetTriggerLastCheck("12345", &lastCheckTest, true)
			So(err, ShouldBeNil)

			count, err = dataBase.GetRemoteChecksUpdatesCount()
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})
	})
}

func TestSelfCheckErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func() {
		count, err := dataBase.GetMetricsUpdatesCount()
		So(count, ShouldEqual, 0)
		So(err, ShouldNotBeNil)

		count, err = dataBase.GetChecksUpdatesCount()
		So(count, ShouldEqual, 0)
		So(err, ShouldNotBeNil)

		err = dataBase.UpdateMetricsHeartbeat()
		So(err, ShouldNotBeNil)
	})
}

func TestNotifierState(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, config)
	emptyDataBase := NewDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	defer dataBase.flush()
	Convey(fmt.Sprintf("Test on empty key '%v'", selfStateNotifierHealth), t, func() {
		Convey("On empty database should return ERROR", func() {
			notifierState, err := emptyDataBase.GetNotifierState()
			So(notifierState, ShouldEqual, selfstate.ERROR)
			So(err, ShouldNotBeNil)
		})
		Convey("On real database should return OK", func() {
			notifierState, err := dataBase.GetNotifierState()
			So(notifierState, ShouldEqual, selfstate.OK)
			So(err, ShouldBeNil)
		})
	})

	Convey(fmt.Sprintf("Test setting '%v' and reading it back", selfStateNotifierHealth), t, func() {
		Convey("Switch notifier to OK", func() {
			err := dataBase.SetNotifierState(selfstate.OK)
			actualNotifierState, err2 := dataBase.GetNotifierState()

			So(actualNotifierState, ShouldEqual, selfstate.OK)
			So(err, ShouldBeNil)
			So(err2, ShouldBeNil)
		})

		Convey("Switch notifier to ERROR", func() {
			err := dataBase.SetNotifierState(selfstate.ERROR)
			actualNotifierState, err2 := dataBase.GetNotifierState()

			So(actualNotifierState, ShouldEqual, selfstate.ERROR)
			So(err, ShouldBeNil)
			So(err2, ShouldBeNil)
		})
	})
}
