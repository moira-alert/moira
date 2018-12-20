package redis

import (
	"testing"

	"github.com/moira-alert/moira/database"
	"github.com/satori/go.uuid"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira/logging/go-logging"
)

func TestTriggerToCheck(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Trigger to check get and add", t, func() {
		triggerID1 := uuid.NewV4().String()
		triggerID2 := uuid.NewV4().String()
		triggerID3 := uuid.NewV4().String()

		actual, err := dataBase.GetTriggerToCheck()
		So(err, ShouldResemble, database.ErrNil)
		So(actual, ShouldBeEmpty)

		count, err := dataBase.GetTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 0)

		err = dataBase.AddTriggersToCheck([]string{triggerID1})
		So(err, ShouldBeNil)

		count, err = dataBase.GetTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 1)

		actual, err = dataBase.GetTriggerToCheck()
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, triggerID1)

		count, err = dataBase.GetTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 0)

		err = dataBase.AddTriggersToCheck([]string{triggerID1})
		So(err, ShouldBeNil)

		err = dataBase.AddTriggersToCheck([]string{triggerID1})
		So(err, ShouldBeNil)

		count, err = dataBase.GetTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 1)

		actual, err = dataBase.GetTriggerToCheck()
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, triggerID1)

		actual, err = dataBase.GetTriggerToCheck()
		So(err, ShouldResemble, database.ErrNil)
		So(actual, ShouldBeEmpty)

		triggerArr := []string{triggerID1, triggerID2, triggerID3}
		err = dataBase.AddTriggersToCheck(triggerArr)
		So(err, ShouldBeNil)

		count, err = dataBase.GetTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 3)

		actual, err = dataBase.GetTriggerToCheck()
		So(err, ShouldBeNil)
		So(actual, ShouldBeIn, triggerArr)
		triggerArr = removeValue(triggerArr, actual)

		actual, err = dataBase.GetTriggerToCheck()
		So(err, ShouldBeNil)
		So(actual, ShouldBeIn, triggerArr)
		triggerArr = removeValue(triggerArr, actual)

		actual, err = dataBase.GetTriggerToCheck()
		So(err, ShouldBeNil)
		So(actual, ShouldBeIn, triggerArr)

		actual, err = dataBase.GetTriggerToCheck()
		So(err, ShouldResemble, database.ErrNil)
		So(actual, ShouldBeEmpty)

		count, err = dataBase.GetTriggersToCheckCount()
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 0)
	})
}

func TestTriggerToCheckConnection(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func() {
		err := dataBase.AddTriggersToCheck([]string{"123"})
		So(err, ShouldNotBeNil)

		triggerID, err := dataBase.GetTriggerToCheck()
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
