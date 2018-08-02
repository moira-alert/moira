package redis

import (
	"testing"

	"github.com/moira-alert/moira/database"
	"github.com/satori/go.uuid"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira/logging/go-logging"
)

func TestRemoteTriggerToCheck(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := NewDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Trigger to check get and add", t, func() {
		triggerID1 := uuid.NewV4().String()
		triggerID2 := uuid.NewV4().String()
		triggerID3 := uuid.NewV4().String()

		actual, err := dataBase.GetRemoteTriggerToCheck()
		So(err, ShouldResemble, database.ErrNil)
		So(actual, ShouldBeEmpty)

		err = dataBase.AddRemoteTriggersToCheck([]string{triggerID1})
		So(err, ShouldBeNil)

		actual, err = dataBase.GetRemoteTriggerToCheck()
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, triggerID1)

		err = dataBase.AddRemoteTriggersToCheck([]string{triggerID1})
		So(err, ShouldBeNil)

		err = dataBase.AddRemoteTriggersToCheck([]string{triggerID1})
		So(err, ShouldBeNil)

		actual, err = dataBase.GetRemoteTriggerToCheck()
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, triggerID1)

		actual, err = dataBase.GetRemoteTriggerToCheck()
		So(err, ShouldResemble, database.ErrNil)
		So(actual, ShouldBeEmpty)

		triggerArr := []string{triggerID1, triggerID2, triggerID3}
		err = dataBase.AddRemoteTriggersToCheck(triggerArr)
		So(err, ShouldBeNil)

		actual, err = dataBase.GetRemoteTriggerToCheck()
		So(err, ShouldBeNil)
		So(actual, ShouldBeIn, triggerArr)
		triggerArr = removeValue(triggerArr, actual)

		actual, err = dataBase.GetRemoteTriggerToCheck()
		So(err, ShouldBeNil)
		So(actual, ShouldBeIn, triggerArr)
		triggerArr = removeValue(triggerArr, actual)

		actual, err = dataBase.GetRemoteTriggerToCheck()
		So(err, ShouldBeNil)
		So(actual, ShouldBeIn, triggerArr)

		actual, err = dataBase.GetRemoteTriggerToCheck()
		So(err, ShouldResemble, database.ErrNil)
		So(actual, ShouldBeEmpty)
	})
}

func TestRemoteTriggerToCheckConnection(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := NewDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func() {
		err := dataBase.AddRemoteTriggersToCheck([]string{"123"})
		So(err, ShouldNotBeNil)

		triggerID, err := dataBase.GetRemoteTriggerToCheck()
		So(triggerID, ShouldBeEmpty)
		So(err, ShouldNotBeNil)
	})
}
