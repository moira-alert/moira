package controller

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/op/go-logging"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetAllTags(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Success", t, func() {
		database.EXPECT().GetTagNames().Return([]string{"_wtf", "atag21", "Tag22", "Hi", "tag1", "1tag"}, nil)
		data, err := GetAllTags(database)
		So(err, ShouldBeNil)
		So(data, ShouldResemble, &dto.TagsData{TagNames: []string{"1tag", "_wtf", "atag21", "Hi", "tag1", "Tag22"}})
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("nooooooooooooooooooooo")
		database.EXPECT().GetTagNames().Return(nil, expected)
		data, err := GetAllTags(database)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(data, ShouldBeNil)
	})
}

func TestDeleteTag(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	tag := "MyTag"

	Convey("Test no trigger ids and subscriptions by tag", t, func() {
		database.EXPECT().GetTagTriggerIDs(tag).Return(nil, nil)
		database.EXPECT().GetTagsSubscriptions([]string{tag}).Return([]*moira.SubscriptionData{}, nil)
		database.EXPECT().RemoveTag(tag).Return(nil)
		resp, err := RemoveTag(database, tag)
		So(err, ShouldBeNil)
		So(resp, ShouldResemble, &dto.MessageResponse{Message: "tag deleted"})
	})

	Convey("Test has trigger ids and subscriptions by tag", t, func() {
		database.EXPECT().GetTagTriggerIDs(tag).Return([]string{"123"}, nil)
		resp, err := RemoveTag(database, tag)
		So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("this tag is assigned to %v triggers. Remove tag from triggers first", 1)))
		So(resp, ShouldBeNil)
	})

	Convey("GetTagTriggerIDs error", t, func() {
		expected := fmt.Errorf("can not read trigger ids")
		database.EXPECT().GetTagTriggerIDs(tag).Return(nil, expected)
		resp, err := RemoveTag(database, tag)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(resp, ShouldBeNil)
	})

	Convey("verification of error handling when receiving subscriptions", t, func() {
		expected := fmt.Errorf("can not read subscriptions")
		database.EXPECT().GetTagTriggerIDs(tag).Return(nil, nil)
		database.EXPECT().GetTagsSubscriptions([]string{tag}).Return(nil, expected)
		resp, err := RemoveTag(database, tag)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(resp, ShouldBeNil)
	})

	Convey("check create error if subscription exists", t, func() {
		data := []*moira.SubscriptionData{{ID: "TestSubscription"}}
		database.EXPECT().GetTagTriggerIDs(tag).Return(nil, nil)
		database.EXPECT().GetTagsSubscriptions([]string{tag}).Return(data, nil)
		resp, err := RemoveTag(database, tag)
		So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("this tag is assigned to 1 subscriptions. Remove tag from subscriptions first")))
		So(resp, ShouldBeNil)
	})

	Convey("verification of error handling during tag removal", t, func() {
		expected := fmt.Errorf("can not delete tag")
		database.EXPECT().GetTagTriggerIDs(tag).Return(nil, nil)
		database.EXPECT().RemoveTag(tag).Return(expected)
		database.EXPECT().GetTagsSubscriptions([]string{tag}).Return(nil, nil)
		resp, err := RemoveTag(database, tag)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(resp, ShouldBeNil)
	})
}

func TestGetAllTagsAndSubscriptions(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")

	Convey("Success get tag stats", t, func() {
		tags := []string{"tag21", "tag22", "tag1"}
		database.EXPECT().GetTagNames().Return(tags, nil)
		database.EXPECT().GetTagsSubscriptions([]string{"tag21"}).Return([]*moira.SubscriptionData{{Tags: []string{"tag21"}}}, nil)
		database.EXPECT().GetTagTriggerIDs("tag21").Return([]string{"trigger21"}, nil)
		database.EXPECT().GetTagsSubscriptions([]string{"tag22"}).Return(make([]*moira.SubscriptionData, 0), nil)
		database.EXPECT().GetTagTriggerIDs("tag22").Return([]string{"trigger22"}, nil)
		database.EXPECT().GetTagsSubscriptions([]string{"tag1"}).Return([]*moira.SubscriptionData{{Tags: []string{"tag1", "tag2"}}}, nil)
		database.EXPECT().GetTagTriggerIDs("tag1").Return(make([]string, 0), nil)
		stat, err := GetAllTagsAndSubscriptions(database, logger)
		So(err, ShouldBeNil)
		So(stat.List, ShouldHaveLength, 3)
		for _, stat := range stat.List {
			if stat.TagName == "tag21" {
				So(stat, ShouldResemble, dto.TagStatistics{TagName: "tag21", Triggers: []string{"trigger21"}, Subscriptions: []moira.SubscriptionData{{Tags: []string{"tag21"}}}})
			}
			if stat.TagName == "tag22" {
				So(stat, ShouldResemble, dto.TagStatistics{TagName: "tag22", Triggers: []string{"trigger22"}, Subscriptions: make([]moira.SubscriptionData, 0)})
			}
			if stat.TagName == "tag1" {
				So(stat, ShouldResemble, dto.TagStatistics{TagName: "tag1", Triggers: make([]string, 0), Subscriptions: []moira.SubscriptionData{{Tags: []string{"tag1", "tag2"}}}})
			}
		}
	})

	Convey("Errors", t, func() {
		Convey("GetTagNames", func() {
			expected := fmt.Errorf("can not get tag names")
			tags := []string{"tag21", "tag22", "tag1"}
			database.EXPECT().GetTagNames().Return(tags, expected)
			stat, err := GetAllTagsAndSubscriptions(database, logger)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(stat, ShouldBeNil)
		})
	})
}
