package redis

import (
	"testing"

	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestLock(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Test lock manipulation", t, func(c C) {
		triggerID1 := "id"

		isSet, err := dataBase.SetTriggerCheckLock(triggerID1)
		c.So(err, ShouldBeNil)
		c.So(isSet, ShouldBeTrue)

		isSet, err = dataBase.SetTriggerCheckLock(triggerID1)
		c.So(err, ShouldBeNil)
		c.So(isSet, ShouldBeFalse)

		err = dataBase.AcquireTriggerCheckLock(triggerID1, 1)
		c.So(err, ShouldNotBeNil)

		err = dataBase.DeleteTriggerCheckLock(triggerID1)
		c.So(err, ShouldBeNil)

		err = dataBase.AcquireTriggerCheckLock(triggerID1, 1)
		c.So(err, ShouldBeNil)

		isSet, err = dataBase.SetTriggerCheckLock(triggerID1)
		c.So(err, ShouldBeNil)
		c.So(isSet, ShouldBeFalse)

		err = dataBase.DeleteTriggerCheckLock(triggerID1)
		c.So(err, ShouldBeNil)
	})
}

func TestLockErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func(c C) {
		err := dataBase.AcquireTriggerCheckLock("tr1", 4)
		c.So(err, ShouldNotBeNil)

		actual, err := dataBase.SetTriggerCheckLock("tr1")
		c.So(err, ShouldNotBeNil)
		c.So(actual, ShouldBeFalse)

		err = dataBase.DeleteTriggerCheckLock("tr1")
		c.So(err, ShouldNotBeNil)
	})
}
