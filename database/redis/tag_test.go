package redis

import (
	"testing"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTagStoring(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()
	client := *dataBase.client

	Convey("Tags manipulation", t, func() {
		trigger := triggers[0]

		Convey("Get tags when they don't exist", func() {
			triggerIDs, err := dataBase.GetTagTriggerIDs(trigger.Tags[0])
			So(err, ShouldBeNil)
			So(triggerIDs, ShouldHaveLength, 0)
			valueStoredAtKey := client.SMembers(dataBase.context, "{moira-tag-triggers}:test-tag-1").Val()
			So(valueStoredAtKey, ShouldBeEmpty)
		})

		Convey("Get tags after the trigger was created with one tag", func() {
			err := dataBase.SaveTrigger(trigger.ID, &trigger)
			So(err, ShouldBeNil)

			tags, err := dataBase.GetTagNames()
			So(err, ShouldBeNil)
			So(tags, ShouldHaveLength, 1)

			triggerIDs, err := dataBase.GetTagTriggerIDs(trigger.Tags[0])
			So(err, ShouldBeNil)
			So(triggerIDs, ShouldHaveLength, 1)
			valueStoredAtKey := client.SMembers(dataBase.context, "{moira-tag-triggers}:test-tag-1").Val()
			So(valueStoredAtKey, ShouldResemble, []string{trigger.ID})
		})

		Convey("Get tags after the only tag of the only trigger was removed", func() {
			err := dataBase.RemoveTag(trigger.Tags[0])
			So(err, ShouldBeNil)

			tags, err := dataBase.GetTagNames()
			So(err, ShouldBeNil)
			So(tags, ShouldHaveLength, 0)

			triggerIDs, err := dataBase.GetTagTriggerIDs(trigger.Tags[0])
			So(err, ShouldBeNil)
			So(triggerIDs, ShouldHaveLength, 0)
		})
	})
}

func TestTagErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabaseWithIncorrectConfig(logger)
	dataBase.Flush()
	defer dataBase.Flush()
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
