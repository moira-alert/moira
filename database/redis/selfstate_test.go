package redis

import (
	"fmt"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSelfCheckWithWritesInChecker(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, config, Checker)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Self state triggers manipulation", t, func(c C) {
		Convey("Empty config", t, func(c C) {
			count, err := dataBase.GetMetricsUpdatesCount()
			c.So(count, ShouldEqual, 0)
			c.So(err, ShouldBeNil)

			count, err = dataBase.GetChecksUpdatesCount()
			c.So(count, ShouldEqual, 0)
			c.So(err, ShouldBeNil)

			count, err = dataBase.GetRemoteChecksUpdatesCount()
			c.So(count, ShouldEqual, 0)
			c.So(err, ShouldBeNil)
		})

		Convey("Update metrics heartbeat test", t, func(c C) {
			err := dataBase.UpdateMetricsHeartbeat()
			c.So(err, ShouldBeNil)

			count, err := dataBase.GetMetricsUpdatesCount()
			c.So(count, ShouldEqual, 1)
			c.So(err, ShouldBeNil)
		})

		Convey("Update metrics checks updates count", t, func(c C) {
			err := dataBase.SetTriggerLastCheck("123", &lastCheckTest, false)
			c.So(err, ShouldBeNil)

			count, err := dataBase.GetChecksUpdatesCount()
			c.So(count, ShouldEqual, 1)
			c.So(err, ShouldBeNil)

			err = dataBase.SetTriggerLastCheck("12345", &lastCheckTest, true)
			c.So(err, ShouldBeNil)

			count, err = dataBase.GetRemoteChecksUpdatesCount()
			c.So(count, ShouldEqual, 1)
			c.So(err, ShouldBeNil)
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
	dataBase := NewDatabase(logger, config, dbSource)
	dataBase.flush()
	defer dataBase.flush()
	Convey(fmt.Sprintf("Self state triggers manipulation in %s", dbSource), t, func(c C) {
		Convey("Update metrics checks updates count", t, func(c C) {
			err := dataBase.SetTriggerLastCheck("123", &lastCheckTest, false)
			c.So(err, ShouldBeNil)

			count, err := dataBase.GetChecksUpdatesCount()
			c.So(count, ShouldEqual, 0)
			c.So(err, ShouldBeNil)

			err = dataBase.SetTriggerLastCheck("12345", &lastCheckTest, true)
			c.So(err, ShouldBeNil)

			count, err = dataBase.GetRemoteChecksUpdatesCount()
			c.So(count, ShouldEqual, 0)
			c.So(err, ShouldBeNil)
		})
	})
}

func TestSelfCheckErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func(c C) {
		count, err := dataBase.GetMetricsUpdatesCount()
		c.So(count, ShouldEqual, 0)
		c.So(err, ShouldNotBeNil)

		count, err = dataBase.GetChecksUpdatesCount()
		c.So(count, ShouldEqual, 0)
		c.So(err, ShouldNotBeNil)

		err = dataBase.UpdateMetricsHeartbeat()
		c.So(err, ShouldNotBeNil)
	})
}

func TestNotifierState(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	emptyDataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey(fmt.Sprintf("Test on empty key '%v'", selfStateNotifierHealth), t, func(c C) {
		Convey("On empty database should return ERROR", t, func(c C) {
			notifierState, err := emptyDataBase.GetNotifierState()
			c.So(notifierState, ShouldEqual, moira.SelfStateERROR)
			c.So(err, ShouldNotBeNil)
		})
		Convey("On real database should return OK", t, func(c C) {
			notifierState, err := dataBase.GetNotifierState()
			c.So(notifierState, ShouldEqual, moira.SelfStateOK)
			c.So(err, ShouldBeNil)
		})
	})

	Convey(fmt.Sprintf("Test setting '%v' and reading it back", selfStateNotifierHealth), t, func(c C) {
		Convey("Switch notifier to OK", t, func(c C) {
			err := dataBase.SetNotifierState(moira.SelfStateOK)
			actualNotifierState, err2 := dataBase.GetNotifierState()

			c.So(actualNotifierState, ShouldEqual, moira.SelfStateOK)
			c.So(err, ShouldBeNil)
			c.So(err2, ShouldBeNil)
		})

		Convey("Switch notifier to ERROR", t, func(c C) {
			err := dataBase.SetNotifierState(moira.SelfStateERROR)
			actualNotifierState, err2 := dataBase.GetNotifierState()

			c.So(actualNotifierState, ShouldEqual, moira.SelfStateERROR)
			c.So(err, ShouldBeNil)
			c.So(err2, ShouldBeNil)
		})
	})
}
