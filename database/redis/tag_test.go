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
	Convey("Tags manipulation", t, func(c C) {
		trigger := triggers[0]
		triggerIDs, err := dataBase.GetTagTriggerIDs(trigger.Tags[0])
		c.So(err, ShouldBeNil)
		c.So(triggerIDs, ShouldHaveLength, 0)

		err = dataBase.SaveTrigger(trigger.ID, &trigger)
		c.So(err, ShouldBeNil)

		tags, err := dataBase.GetTagNames()
		c.So(err, ShouldBeNil)
		c.So(tags, ShouldHaveLength, 1)

		triggerIDs, err = dataBase.GetTagTriggerIDs(trigger.Tags[0])
		c.So(err, ShouldBeNil)
		c.So(triggerIDs, ShouldHaveLength, 1)

		err = dataBase.RemoveTag(trigger.Tags[0])
		c.So(err, ShouldBeNil)

		tags, err = dataBase.GetTagNames()
		c.So(err, ShouldBeNil)
		c.So(tags, ShouldHaveLength, 0)

		triggerIDs, err = dataBase.GetTagTriggerIDs(trigger.Tags[0])
		c.So(err, ShouldBeNil)
		c.So(triggerIDs, ShouldHaveLength, 0)
	})
}

func TestTagErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func(c C) {
		actual, err := dataBase.GetTagNames()
		c.So(err, ShouldNotBeNil)
		c.So(actual, ShouldBeNil)

		err = dataBase.RemoveTag("ds")
		c.So(err, ShouldNotBeNil)

		actual, err = dataBase.GetTagTriggerIDs("34")
		c.So(err, ShouldNotBeNil)
		c.So(actual, ShouldBeNil)
	})
}
