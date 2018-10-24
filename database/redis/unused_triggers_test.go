package redis

import (
	"testing"

	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUnusedTriggers(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := NewDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Check marking unused", t, func() {
		// Mark trigger 123 unused
		err := dataBase.MarkTriggersAsUnused("123")
		So(err, ShouldBeNil)

		triggerIDs, err := dataBase.GetUnusedTriggerIDs()
		So(triggerIDs, ShouldResemble, []string{"123"})
		So(err, ShouldBeNil)

		// Repeat procedure till success
		err = dataBase.MarkTriggersAsUnused("123")
		So(err, ShouldBeNil)

		triggerIDs, err = dataBase.GetUnusedTriggerIDs()
		So(triggerIDs, ShouldResemble, []string{"123"})
		So(err, ShouldBeNil)

		// Trying to unmark it
		err = dataBase.MarkTriggersAsUsed("123")
		So(err, ShouldBeNil)

		triggerIDs, err = dataBase.GetUnusedTriggerIDs()
		So(triggerIDs, ShouldBeEmpty)
		So(err, ShouldBeNil)

		// Ok, let's raise the rates
		err = dataBase.MarkTriggersAsUnused("123", "234", "345")
		So(err, ShouldBeNil)

		triggerIDs, err = dataBase.GetUnusedTriggerIDs()
		So(triggerIDs, ShouldResemble, []string{"123", "234", "345"})
		So(err, ShouldBeNil)

		// But, maybe we want to see the world burn?
		err = dataBase.MarkTriggersAsUnused("123", "234", "345")
		So(err, ShouldBeNil)

		triggerIDs, err = dataBase.GetUnusedTriggerIDs()
		So(triggerIDs, ShouldResemble, []string{"123", "234", "345"})
		So(err, ShouldBeNil)

		err = dataBase.MarkTriggersAsUsed("123", "234")
		So(err, ShouldBeNil)

		triggerIDs, err = dataBase.GetUnusedTriggerIDs()
		So(triggerIDs, ShouldResemble, []string{"345"})
		So(err, ShouldBeNil)

		// Okey, I want to unmark non-existing triggers
		err = dataBase.MarkTriggersAsUsed("alalala", "babababa")
		So(err, ShouldBeNil)

		triggerIDs, err = dataBase.GetUnusedTriggerIDs()
		So(triggerIDs, ShouldResemble, []string{"345"})
		So(err, ShouldBeNil)

		// AAAAND magic
		err = dataBase.MarkTriggersAsUsed()
		So(err, ShouldBeNil)

		err = dataBase.MarkTriggersAsUnused()
		So(err, ShouldBeNil)
	})
}

func TestUnusedTriggersConnection(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := NewDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func() {
		err := dataBase.MarkTriggersAsUnused("123")
		So(err, ShouldNotBeNil)

		triggerIDs, err := dataBase.GetUnusedTriggerIDs()
		So(triggerIDs, ShouldBeEmpty)
		So(err, ShouldNotBeNil)
	})
}
