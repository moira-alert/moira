package redis

import (
	"testing"

	"github.com/gofrs/uuid"
	. "github.com/smartystreets/goconvey/convey"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
)

func TestTriggerToCheck(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test", true)
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Trigger to check get and add", t, func() {
		triggerID1 := uuid.Must(uuid.NewV4()).String()
		triggerID2 := uuid.Must(uuid.NewV4()).String()
		triggerID3 := uuid.Must(uuid.NewV4()).String()
		triggerID4 := uuid.Must(uuid.NewV4()).String()
		triggerID5 := uuid.Must(uuid.NewV4()).String()
		triggerID6 := uuid.Must(uuid.NewV4()).String()

		actual, err := dataBase.GetLocalTriggersToCheck(1)
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		count, err := dataBase.GetLocalTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 0)

		err = dataBase.AddLocalTriggersToCheck([]string{triggerID1})
		So(err, ShouldBeNil)

		count, err = dataBase.GetLocalTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 1)

		actual, err = dataBase.GetLocalTriggersToCheck(1)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, []string{triggerID1})

		count, err = dataBase.GetLocalTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 0)

		err = dataBase.AddLocalTriggersToCheck([]string{triggerID1})
		So(err, ShouldBeNil)

		err = dataBase.AddLocalTriggersToCheck([]string{triggerID1})
		So(err, ShouldBeNil)

		count, err = dataBase.GetLocalTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 1)

		actual, err = dataBase.GetLocalTriggersToCheck(1)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, []string{triggerID1})

		actual, err = dataBase.GetLocalTriggersToCheck(1)
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		triggerArr := []string{triggerID1, triggerID2, triggerID3, triggerID4, triggerID5, triggerID6}
		err = dataBase.AddLocalTriggersToCheck(triggerArr)
		So(err, ShouldBeNil)

		count, err = dataBase.GetLocalTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 6)

		actual, err = dataBase.GetLocalTriggersToCheck(1)
		So(err, ShouldBeNil)
		So(actual, ShouldHaveLength, 1)
		So(actual[0], ShouldBeIn, triggerArr)
		triggerArr = removeValue(triggerArr, actual[0])

		actual, err = dataBase.GetLocalTriggersToCheck(2)
		So(err, ShouldBeNil)
		So(actual, ShouldHaveLength, 2)
		So(actual[0], ShouldBeIn, triggerArr)
		So(actual[1], ShouldBeIn, triggerArr)
		triggerArr = removeValue(triggerArr, actual[0])
		triggerArr = removeValue(triggerArr, actual[1])

		actual, err = dataBase.GetLocalTriggersToCheck(6)
		So(err, ShouldBeNil)
		So(actual, ShouldHaveLength, 3)
		So(actual[0], ShouldBeIn, triggerArr)
		So(actual[1], ShouldBeIn, triggerArr)
		So(actual[2], ShouldBeIn, triggerArr)

		actual, err = dataBase.GetLocalTriggersToCheck(1)
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		count, err = dataBase.GetLocalTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 0)
	})
}

func TestRemoteTriggerToCheck(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test", true)
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Trigger to check get and add", t, func() {
		triggerID1 := uuid.Must(uuid.NewV4()).String()
		triggerID2 := uuid.Must(uuid.NewV4()).String()
		triggerID3 := uuid.Must(uuid.NewV4()).String()
		triggerID4 := uuid.Must(uuid.NewV4()).String()
		triggerID5 := uuid.Must(uuid.NewV4()).String()
		triggerID6 := uuid.Must(uuid.NewV4()).String()

		actual, err := dataBase.GetRemoteTriggersToCheck(1)
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		count, err := dataBase.GetRemoteTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 0)

		err = dataBase.AddRemoteTriggersToCheck([]string{triggerID1})
		So(err, ShouldBeNil)

		count, err = dataBase.GetRemoteTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 1)

		actual, err = dataBase.GetRemoteTriggersToCheck(1)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, []string{triggerID1})

		count, err = dataBase.GetRemoteTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 0)

		err = dataBase.AddRemoteTriggersToCheck([]string{triggerID1})
		So(err, ShouldBeNil)

		err = dataBase.AddRemoteTriggersToCheck([]string{triggerID1})
		So(err, ShouldBeNil)

		count, err = dataBase.GetRemoteTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 1)

		actual, err = dataBase.GetRemoteTriggersToCheck(1)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, []string{triggerID1})

		actual, err = dataBase.GetRemoteTriggersToCheck(1)
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		triggerArr := []string{triggerID1, triggerID2, triggerID3, triggerID4, triggerID5, triggerID6}
		err = dataBase.AddRemoteTriggersToCheck(triggerArr)
		So(err, ShouldBeNil)

		count, err = dataBase.GetRemoteTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 6)

		actual, err = dataBase.GetRemoteTriggersToCheck(1)
		So(err, ShouldBeNil)
		So(actual[0], ShouldBeIn, triggerArr)
		triggerArr = removeValue(triggerArr, actual[0])

		actual, err = dataBase.GetRemoteTriggersToCheck(2)
		So(err, ShouldBeNil)
		So(actual, ShouldHaveLength, 2)
		So(actual[0], ShouldBeIn, triggerArr)
		So(actual[1], ShouldBeIn, triggerArr)
		triggerArr = removeValue(triggerArr, actual[0])
		triggerArr = removeValue(triggerArr, actual[1])

		actual, err = dataBase.GetRemoteTriggersToCheck(6)
		So(err, ShouldBeNil)
		So(actual, ShouldHaveLength, 3)
		So(actual[0], ShouldBeIn, triggerArr)
		So(actual[1], ShouldBeIn, triggerArr)
		So(actual[2], ShouldBeIn, triggerArr)

		actual, err = dataBase.GetRemoteTriggersToCheck(5)
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		count, err = dataBase.GetLocalTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 0)
	})
}

func TestRemoteTriggerToCheckConnection(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test", true)
	dataBase := newTestDatabase(logger, incorrectConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func() {
		err := dataBase.AddRemoteTriggersToCheck([]string{"123"})
		So(err, ShouldNotBeNil)

		triggerID, err := dataBase.GetRemoteTriggersToCheck(1)
		So(triggerID, ShouldBeEmpty)
		So(err, ShouldNotBeNil)
	})
}

func TestTriggerToCheckConnection(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test", true)
	dataBase := newTestDatabase(logger, incorrectConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func() {
		err := dataBase.AddLocalTriggersToCheck([]string{"123"})
		So(err, ShouldNotBeNil)

		triggerID, err := dataBase.GetLocalTriggersToCheck(1)
		So(triggerID, ShouldBeEmpty)
		So(err, ShouldNotBeNil)
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
