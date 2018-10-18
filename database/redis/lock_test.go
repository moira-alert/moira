package redis

import (
	"fmt"
	"testing"
	"time"

	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestLock(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Test lock manipulation", t, func() {
		triggerID1 := "id"

		isSet, err := dataBase.SetTriggerCheckLock(triggerID1)
		So(err, ShouldBeNil)
		So(isSet, ShouldBeTrue)

		isSet, err = dataBase.SetTriggerCheckLock(triggerID1)
		So(err, ShouldBeNil)
		So(isSet, ShouldBeFalse)

		err = dataBase.AcquireTriggerCheckLock(triggerID1, 1)
		So(err, ShouldNotBeNil)

		err = dataBase.DeleteTriggerCheckLock(triggerID1)
		So(err, ShouldBeNil)

		err = dataBase.AcquireTriggerCheckLock(triggerID1, 1)
		So(err, ShouldBeNil)

		isSet, err = dataBase.SetTriggerCheckLock(triggerID1)
		So(err, ShouldBeNil)
		So(isSet, ShouldBeFalse)

		err = dataBase.DeleteTriggerCheckLock(triggerID1)
		So(err, ShouldBeNil)
	})
}

func TestLockErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func() {
		err := dataBase.AcquireTriggerCheckLock("tr1", 4)
		So(err, ShouldNotBeNil)

		actual, err := dataBase.SetTriggerCheckLock("tr1")
		So(err, ShouldNotBeNil)
		So(actual, ShouldBeFalse)

		err = dataBase.DeleteTriggerCheckLock("tr1")
		So(err, ShouldNotBeNil)
	})
}

func testLockWithTTLExpireErrorExpected(lockTTL int, lockWindow int, locker func() bool) []bool {
	// This test takes ttl expire error into account
	// https://redis.io/commands/expire#expire-accuracy
	//
	// So both example outputs are possible:
	//
	// 2018-10-11 13:34:06.4726686 +0500 +05 m=+4.026959900 Attempt: success
	// 2018-10-11 13:34:06.4738116 +0500 +05 m=+4.028102800 Attempt: failure
	// 2018-10-11 13:34:06.4749116 +0500 +05 m=+4.029202900 Attempt: failure
	//
	// 2018-10-11 13:31:24.0643861 +0500 +05 m=+4.045829100 Attempt: failure
	// 2018-10-11 13:31:24.0664404 +0500 +05 m=+4.047883400 Attempt: success
	// 2018-10-11 13:31:24.0675413 +0500 +05 m=+4.048984300 Attempt: failure
	resultMap := map[bool]string{
		true:  "success",
		false: "failure",
	}
	time.Sleep(time.Duration(lockTTL-1) * time.Millisecond)
	lockExpiryTicker := time.NewTicker(time.Millisecond)
	lockResults := make([]bool, 0)
	for t := range lockExpiryTicker.C {
		result := locker()
		fmt.Printf("%s Attempt: %s\n", t.String(), resultMap[result])
		lockResults = append(lockResults, result)
		if len(lockResults) == lockWindow {
			break
		}
	}
	return lockResults
}
