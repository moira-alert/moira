package redis

import (
	"testing"
	"time"

	"github.com/moira-alert/moira/logging/go-logging"
	"github.com/satori/go.uuid"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTriggersToUpdate(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := NewDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Trigger to update add and fetch", t, func() {
		triggerID1 := uuid.NewV4().String()
		triggerID2 := uuid.NewV4().String()
		triggerID3 := uuid.NewV4().String()

		actual, err := dataBase.FetchTriggersToUpdate(time.Now().Unix())
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		startTime := time.Now().Unix()

		// current time ≈ startTime + 1
		time.Sleep(time.Second)
		err = dataBase.AddTriggersToUpdate(triggerID1)
		So(err, ShouldBeNil)

		//current time ≈ startTime + 2
		time.Sleep(time.Second)
		actual, err = dataBase.FetchTriggersToUpdate(time.Now().Unix())
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		actual, err = dataBase.FetchTriggersToUpdate(startTime)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, []string{triggerID1})

		//current time ≈ startTime + 3
		time.Sleep(time.Second)
		err = dataBase.AddTriggersToUpdate(triggerID2, triggerID3)
		So(err, ShouldBeNil)

		actual, err = dataBase.FetchTriggersToUpdate(startTime)
		So(err, ShouldBeNil)
		So(actual, ShouldHaveLength, 3)

		err = dataBase.RemoveTriggersToUpdate(startTime + 2)
		So(err, ShouldBeNil)

		actual, err = dataBase.FetchTriggersToUpdate(startTime)
		So(err, ShouldBeNil)
		So(actual, ShouldHaveLength, 2)

		err = dataBase.RemoveTriggersToUpdate(startTime + 4)
		So(err, ShouldBeNil)

		actual, err = dataBase.FetchTriggersToUpdate(startTime)
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)
	})
}

func TestTriggerToUpdateConnection(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := NewDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Should throw error when no connection", t, func() {
		err := dataBase.AddTriggersToUpdate("123")
		So(err, ShouldNotBeNil)

		triggerID, err := dataBase.FetchTriggersToUpdate(time.Now().Unix())
		So(triggerID, ShouldBeEmpty)
		So(err, ShouldNotBeNil)
	})
}
