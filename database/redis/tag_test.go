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

func TestAddTag(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "warn", "test", true)
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	Convey("When AddTag was called", t, func() {
		client := *dataBase.client

		const tag = "tag"

		err := dataBase.AddTag(tag)
		So(err, ShouldBeNil)

		Convey("Tag with trigger should be and abandoned tag shouldn't be in database ", func() {
			isExists, err := client.SIsMember(dataBase.context, "moira-tags", tag).Result()
			So(err, ShouldBeNil)
			So(isExists, ShouldBeTrue)
		})
	})
}

func TestCleanUpAbandonedTags(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "warn", "test", true)
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	Convey("Given tag with trigger and abandoned tag (tag without trigger)", t, func() {
		client := *dataBase.client

		const (
			triggerID         = "triggerID"
			tagWithTrigger    = "tagWithTrigger"
			tagWithoutTrigger = "tagWithoutTrigger"
		)

		client.SAdd(dataBase.context, tagTriggersKey(tagWithTrigger), triggerID)
		err := dataBase.AddTag(tagWithTrigger)
		So(err, ShouldBeNil)
		err = dataBase.AddTag(tagWithoutTrigger)
		So(err, ShouldBeNil)

		isExists, err := client.SIsMember(dataBase.context, "{moira-tag-triggers}:"+tagWithTrigger, triggerID).Result()
		So(err, ShouldBeNil)
		So(isExists, ShouldBeTrue)
		isExists, err = client.SIsMember(dataBase.context, "moira-tags", tagWithTrigger).Result()
		So(err, ShouldBeNil)
		So(isExists, ShouldBeTrue)
		isExists, err = client.SIsMember(dataBase.context, "moira-tags", tagWithoutTrigger).Result()
		So(err, ShouldBeNil)
		So(isExists, ShouldBeTrue)

		Convey("When clean up tags was called", func() {
			count, err := dataBase.CleanUpAbandonedTags()
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 1)

			Convey("Tag with trigger should be and abandoned tag shouldn't be in database ", func() {
				isExists, err = client.SIsMember(dataBase.context, "moira-tags", tagWithoutTrigger).Result()
				So(err, ShouldBeNil)
				So(isExists, ShouldBeFalse)

				isExists, err = client.SIsMember(dataBase.context, "{moira-tag-triggers}:"+tagWithTrigger, triggerID).Result()
				So(err, ShouldBeNil)
				So(isExists, ShouldBeTrue)

				isExists, err = client.SIsMember(dataBase.context, "moira-tags", tagWithTrigger).Result()
				So(err, ShouldBeNil)
				So(isExists, ShouldBeTrue)
			})
		})
	})
}
