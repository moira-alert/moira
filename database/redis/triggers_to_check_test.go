package redis

import (
	"testing"

	"github.com/gofrs/uuid"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira/logging/go-logging"
)

func TestTriggerToCheck(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Trigger to check get and add", t, func(c C) {
		triggerID1 := uuid.Must(uuid.NewV4()).String()
		triggerID2 := uuid.Must(uuid.NewV4()).String()
		triggerID3 := uuid.Must(uuid.NewV4()).String()
		triggerID4 := uuid.Must(uuid.NewV4()).String()
		triggerID5 := uuid.Must(uuid.NewV4()).String()
		triggerID6 := uuid.Must(uuid.NewV4()).String()

		actual, err := dataBase.GetLocalTriggersToCheck(1)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldBeEmpty)

		count, err := dataBase.GetLocalTriggersToCheckCount()
		c.So(err, ShouldBeNil)
		c.So(count, ShouldEqual, 0)

		err = dataBase.AddLocalTriggersToCheck([]string{triggerID1})
		c.So(err, ShouldBeNil)

		count, err = dataBase.GetLocalTriggersToCheckCount()
		c.So(err, ShouldBeNil)
		c.So(count, ShouldEqual, 1)

		actual, err = dataBase.GetLocalTriggersToCheck(1)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldResemble, []string{triggerID1})

		count, err = dataBase.GetLocalTriggersToCheckCount()
		c.So(err, ShouldBeNil)
		c.So(count, ShouldEqual, 0)

		err = dataBase.AddLocalTriggersToCheck([]string{triggerID1})
		c.So(err, ShouldBeNil)

		err = dataBase.AddLocalTriggersToCheck([]string{triggerID1})
		c.So(err, ShouldBeNil)

		count, err = dataBase.GetLocalTriggersToCheckCount()
		c.So(err, ShouldBeNil)
		c.So(count, ShouldEqual, 1)

		actual, err = dataBase.GetLocalTriggersToCheck(1)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldResemble, []string{triggerID1})

		actual, err = dataBase.GetLocalTriggersToCheck(1)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldBeEmpty)

		triggerArr := []string{triggerID1, triggerID2, triggerID3, triggerID4, triggerID5, triggerID6}
		err = dataBase.AddLocalTriggersToCheck(triggerArr)
		c.So(err, ShouldBeNil)

		count, err = dataBase.GetLocalTriggersToCheckCount()
		c.So(err, ShouldBeNil)
		c.So(count, ShouldEqual, 6)

		actual, err = dataBase.GetLocalTriggersToCheck(1)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldHaveLength, 1)
		c.So(actual[0], ShouldBeIn, triggerArr)
		triggerArr = removeValue(triggerArr, actual[0])

		actual, err = dataBase.GetLocalTriggersToCheck(2)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldHaveLength, 2)
		c.So(actual[0], ShouldBeIn, triggerArr)
		c.So(actual[1], ShouldBeIn, triggerArr)
		triggerArr = removeValue(triggerArr, actual[0])
		triggerArr = removeValue(triggerArr, actual[1])

		actual, err = dataBase.GetLocalTriggersToCheck(6)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldHaveLength, 3)
		c.So(actual[0], ShouldBeIn, triggerArr)
		c.So(actual[1], ShouldBeIn, triggerArr)
		c.So(actual[2], ShouldBeIn, triggerArr)

		actual, err = dataBase.GetLocalTriggersToCheck(1)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldBeEmpty)

		count, err = dataBase.GetLocalTriggersToCheckCount()
		c.So(err, ShouldBeNil)
		c.So(count, ShouldEqual, 0)
	})
}

func TestRemoteTriggerToCheck(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Trigger to check get and add", t, func(c C) {
		triggerID1 := uuid.Must(uuid.NewV4()).String()
		triggerID2 := uuid.Must(uuid.NewV4()).String()
		triggerID3 := uuid.Must(uuid.NewV4()).String()
		triggerID4 := uuid.Must(uuid.NewV4()).String()
		triggerID5 := uuid.Must(uuid.NewV4()).String()
		triggerID6 := uuid.Must(uuid.NewV4()).String()

		actual, err := dataBase.GetRemoteTriggersToCheck(1)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldBeEmpty)

		count, err := dataBase.GetRemoteTriggersToCheckCount()
		c.So(err, ShouldBeNil)
		c.So(count, ShouldEqual, 0)

		err = dataBase.AddRemoteTriggersToCheck([]string{triggerID1})
		c.So(err, ShouldBeNil)

		count, err = dataBase.GetRemoteTriggersToCheckCount()
		c.So(err, ShouldBeNil)
		c.So(count, ShouldEqual, 1)

		actual, err = dataBase.GetRemoteTriggersToCheck(1)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldResemble, []string{triggerID1})

		count, err = dataBase.GetRemoteTriggersToCheckCount()
		c.So(err, ShouldBeNil)
		c.So(count, ShouldEqual, 0)

		err = dataBase.AddRemoteTriggersToCheck([]string{triggerID1})
		c.So(err, ShouldBeNil)

		err = dataBase.AddRemoteTriggersToCheck([]string{triggerID1})
		c.So(err, ShouldBeNil)

		count, err = dataBase.GetRemoteTriggersToCheckCount()
		c.So(err, ShouldBeNil)
		c.So(count, ShouldEqual, 1)

		actual, err = dataBase.GetRemoteTriggersToCheck(1)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldResemble, []string{triggerID1})

		actual, err = dataBase.GetRemoteTriggersToCheck(1)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldBeEmpty)

		triggerArr := []string{triggerID1, triggerID2, triggerID3, triggerID4, triggerID5, triggerID6}
		err = dataBase.AddRemoteTriggersToCheck(triggerArr)
		c.So(err, ShouldBeNil)

		count, err = dataBase.GetRemoteTriggersToCheckCount()
		c.So(err, ShouldBeNil)
		c.So(count, ShouldEqual, 6)

		actual, err = dataBase.GetRemoteTriggersToCheck(1)
		c.So(err, ShouldBeNil)
		c.So(actual[0], ShouldBeIn, triggerArr)
		triggerArr = removeValue(triggerArr, actual[0])

		actual, err = dataBase.GetRemoteTriggersToCheck(2)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldHaveLength, 2)
		c.So(actual[0], ShouldBeIn, triggerArr)
		c.So(actual[1], ShouldBeIn, triggerArr)
		triggerArr = removeValue(triggerArr, actual[0])
		triggerArr = removeValue(triggerArr, actual[1])

		actual, err = dataBase.GetRemoteTriggersToCheck(6)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldHaveLength, 3)
		c.So(actual[0], ShouldBeIn, triggerArr)
		c.So(actual[1], ShouldBeIn, triggerArr)
		c.So(actual[2], ShouldBeIn, triggerArr)

		actual, err = dataBase.GetRemoteTriggersToCheck(5)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldBeEmpty)

		count, err = dataBase.GetLocalTriggersToCheckCount()
		c.So(err, ShouldBeNil)
		c.So(count, ShouldEqual, 0)
	})
}

func TestRemoteTriggerToCheckConnection(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func(c C) {
		err := dataBase.AddRemoteTriggersToCheck([]string{"123"})
		c.So(err, ShouldNotBeNil)

		triggerID, err := dataBase.GetRemoteTriggersToCheck(1)
		c.So(triggerID, ShouldBeEmpty)
		c.So(err, ShouldNotBeNil)
	})
}

func TestTriggerToCheckConnection(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func(c C) {
		err := dataBase.AddLocalTriggersToCheck([]string{"123"})
		c.So(err, ShouldNotBeNil)

		triggerID, err := dataBase.GetLocalTriggersToCheck(1)
		c.So(triggerID, ShouldBeEmpty)
		c.So(err, ShouldNotBeNil)
	})
}

func removeValue(triggerArr []string, triggerID string) []string {
	index := 0
	for i, trigger := range triggerArr {
		if trigger == triggerID {
			index = i
			break
		}
	}
	return append(triggerArr[:index], triggerArr[index+1:]...)
}
