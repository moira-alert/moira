package redis

import (
	"testing"

	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTagStoring(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Tags manipulation", t, func() {
		trigger := triggers[0]
		triggerIDs, err := dataBase.GetTagTriggerIDs(trigger.Tags[0])
		So(err, ShouldBeNil)
		So(triggerIDs, ShouldHaveLength, 0)

		err = dataBase.SaveTrigger(trigger.ID, &trigger)
		So(err, ShouldBeNil)

		tags, err := dataBase.GetTagNames()
		So(err, ShouldBeNil)
		So(tags, ShouldHaveLength, 2)

		triggerIDs, err = dataBase.GetTagTriggerIDs(trigger.Tags[0])
		So(err, ShouldBeNil)
		So(triggerIDs, ShouldHaveLength, 1)

		err = dataBase.RemoveTag(trigger.Tags[0])
		So(err, ShouldBeNil)

		tags, err = dataBase.GetTagNames()
		So(err, ShouldBeNil)
		So(tags, ShouldHaveLength, 1)

		triggerIDs, err = dataBase.GetTagTriggerIDs(trigger.Tags[0])
		So(err, ShouldBeNil)
		So(triggerIDs, ShouldHaveLength, 0)
	})
}

func TestTagErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func() {
		actual, err := dataBase.GetTagNames()
		So(err, ShouldNotBeNil)
		So(actual, ShouldBeNil)

		err = dataBase.RemoveTag("ds")
		So(err, ShouldNotBeNil)

		actual, err = dataBase.GetTagTriggerIDs("34")
		So(err, ShouldNotBeNil)
		So(actual, ShouldBeNil)
	})
}
