package redis

import (
	"fmt"
	"testing"

	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSelfCheckWithWritesInChecker(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.source = Checker
	dataBase.Flush()
	defer dataBase.Flush()
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
			err := dataBase.SetTriggerLastCheck("123", &lastCheckTest, moira.GraphiteLocal)
			So(err, ShouldBeNil)

			count, err := dataBase.GetChecksUpdatesCount()
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)

			err = dataBase.SetTriggerLastCheck("12345", &lastCheckTest, moira.GraphiteRemote)
			So(err, ShouldBeNil)

			count, err = dataBase.GetRemoteChecksUpdatesCount()
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})
	})
}

func TestSelfCheckWithWritesNotInChecker(t *testing.T) {
	dbSources := []DBSource{Filter, API, Notifier, Cli, testSource}
	for _, dbSource := range dbSources {
		testSelfCheckWithWritesInDBSource(t, dbSource)
	}
}

func testSelfCheckWithWritesInDBSource(t *testing.T, dbSource DBSource) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.source = dbSource
	dataBase.Flush()
	defer dataBase.Flush()
	Convey(fmt.Sprintf("Self state triggers manipulation in %s", dbSource), t, func() {
		Convey("Update metrics checks updates count", func() {
			err := dataBase.SetTriggerLastCheck("123", &lastCheckTest, moira.GraphiteLocal)
			So(err, ShouldBeNil)

			count, err := dataBase.GetChecksUpdatesCount()
			So(count, ShouldEqual, 0)
			So(err, ShouldBeNil)

			err = dataBase.SetTriggerLastCheck("12345", &lastCheckTest, moira.GraphiteRemote)
			So(err, ShouldBeNil)

			count, err = dataBase.GetRemoteChecksUpdatesCount()
			So(count, ShouldEqual, 0)
			So(err, ShouldBeNil)
		})
	})
}

func TestSelfCheckErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabaseWithIncorrectConfig(logger)
	dataBase.Flush()
	defer dataBase.Flush()
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
	dataBase := NewTestDatabase(logger)
	emptyDataBase := NewTestDatabaseWithIncorrectConfig(logger)
	dataBase.Flush()
	defer dataBase.Flush()
	Convey(fmt.Sprintf("Test on empty key '%v'", selfStateNotifierHealth), t, func() {
		Convey("On empty database should return ERROR", func() {
			notifierState, err := emptyDataBase.GetNotifierState()
			So(notifierState, ShouldEqual, moira.SelfStateERROR)
			So(err, ShouldNotBeNil)
		})
		Convey("On real database should return OK", func() {
			notifierState, err := dataBase.GetNotifierState()
			So(notifierState, ShouldEqual, moira.SelfStateOK)
			So(err, ShouldBeNil)
		})
	})

	Convey(fmt.Sprintf("Test setting '%v' and reading it back", selfStateNotifierHealth), t, func() {
		Convey("Switch notifier to OK", func() {
			err := dataBase.SetNotifierState(moira.SelfStateOK)
			actualNotifierState, err2 := dataBase.GetNotifierState()

			So(actualNotifierState, ShouldEqual, moira.SelfStateOK)
			So(err, ShouldBeNil)
			So(err2, ShouldBeNil)
		})

		Convey("Switch notifier to ERROR", func() {
			err := dataBase.SetNotifierState(moira.SelfStateERROR)
			actualNotifierState, err2 := dataBase.GetNotifierState()

			So(actualNotifierState, ShouldEqual, moira.SelfStateERROR)
			So(err, ShouldBeNil)
			So(err2, ShouldBeNil)
		})
	})
}
