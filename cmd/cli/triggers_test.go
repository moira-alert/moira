package main

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mocks "github.com/moira-alert/moira/mock/moira-alert"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_deleteTriggers(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	db := mocks.NewMockDatabase(mockCtrl)
	delay = 1 * time.Millisecond

	Convey("Success delete triggers", t, func() {
		db.EXPECT().RemoveTrigger("trigger-1").Return(nil)
		db.EXPECT().RemoveTrigger("trigger-2").Return(nil)

		triggersToDelete := []string{"trigger-1", "trigger-2"}
		err := deleteTriggers(logger, triggersToDelete, db)
		So(err, ShouldBeNil)
	})

	Convey("Cannot delete trigger-2", t, func() {
		db.EXPECT().RemoveTrigger("trigger-1").Return(nil)
		db.EXPECT().RemoveTrigger("trigger-2").Return(errors.New("oops"))

		triggersToDelete := []string{"trigger-1", "trigger-2"}
		err := deleteTriggers(logger, triggersToDelete, db)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldResemble, "can't remove trigger with id trigger-2: oops")
	})
}

func Test_handleRemoveTriggersStartWith(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	db := mocks.NewMockDatabase(mockCtrl)
	delay = 1 * time.Millisecond

	Convey("Success delete triggers", t, func() {
		triggersToDelete := []string{"trigger-1", "trigger-2"}

		db.EXPECT().GetTriggerIDsStartWith("trigger").Return(triggersToDelete, nil)
		db.EXPECT().RemoveTrigger("trigger-1").Return(nil)
		db.EXPECT().RemoveTrigger("trigger-2").Return(nil)

		err := handleRemoveTriggersStartWith(logger, db, "trigger")
		So(err, ShouldBeNil)
	})

	Convey("Cannot get GetTriggerIDsStartWith", t, func() {
		db.EXPECT().GetTriggerIDsStartWith("trigger").Return(nil, errors.New("oops"))

		err := handleRemoveTriggersStartWith(logger, db, "trigger")
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldResemble, "can't get trigger IDs start with prefix trigger: oops")
	})
}

func Test_handleRemoveUnusedTriggersStartWith(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	db := mocks.NewMockDatabase(mockCtrl)
	delay = 1 * time.Millisecond

	Convey("Success delete triggers", t, func() {
		triggers := []string{"trigger-1", "trigger-2"}

		db.EXPECT().GetTriggerIDsStartWith("trigger").Return(triggers, nil)
		db.EXPECT().GetUnusedTriggerIDs().Return([]string{"trigger-1"}, nil)
		db.EXPECT().RemoveTrigger("trigger-1").Return(nil)

		err := handleRemoveUnusedTriggersStartWith(logger, db, "trigger")
		So(err, ShouldBeNil)
	})

	Convey("Cannot get GetTriggerIDsStartWith", t, func() {
		db.EXPECT().GetTriggerIDsStartWith("trigger").Return(nil, errors.New("oops"))

		err := handleRemoveUnusedTriggersStartWith(logger, db, "trigger")
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldResemble, "can't get trigger IDs start with prefix trigger: oops")
	})

	Convey("Cannot get GetTriggerIDsStartWith", t, func() {
		db.EXPECT().GetTriggerIDsStartWith("trigger").Return([]string{"trigger-1"}, nil)
		db.EXPECT().GetUnusedTriggerIDs().Return(nil, errors.New("oops"))

		err := handleRemoveUnusedTriggersStartWith(logger, db, "trigger")
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldResemble, "can't get unused trigger IDs; err: oops")
	})
}

func Test_handleRemoveUnusedTriggersWithTTL(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	db := mocks.NewMockDatabase(mockCtrl)
	delay = 1 * time.Millisecond
	nowTime := time.Now()

	Convey("Success delete triggers: updated at is set", t, func() {
		updatedAt := nowTime.Add(-24 * time.Hour).Unix()
		db.EXPECT().GetTrigger("trigger-1").Return(moira.Trigger{UpdatedAt: &updatedAt}, nil)
		db.EXPECT().GetUnusedTriggerIDs().Return([]string{"trigger-1"}, nil)
		db.EXPECT().RemoveTrigger("trigger-1").Return(nil)

		ttl := int64(2 * time.Hour.Seconds())
		err := handleRemoveUnusedTriggersWithTTL(logger, db, ttl)
		So(err, ShouldBeNil)
	})

	Convey("Success delete triggers: created at is set", t, func() {
		createdAt := nowTime.Add(-24 * time.Hour).Unix()
		db.EXPECT().GetTrigger("trigger-1").Return(moira.Trigger{CreatedAt: &createdAt}, nil)
		db.EXPECT().GetUnusedTriggerIDs().Return([]string{"trigger-1"}, nil)
		db.EXPECT().RemoveTrigger("trigger-1").Return(nil)

		ttl := int64(2 * time.Hour.Seconds())
		err := handleRemoveUnusedTriggersWithTTL(logger, db, ttl)
		So(err, ShouldBeNil)
	})

	Convey("Success delete triggers: created at and updated_at is no set", t, func() {
		db.EXPECT().GetTrigger("trigger-1").Return(moira.Trigger{}, nil)
		db.EXPECT().GetUnusedTriggerIDs().Return([]string{"trigger-1"}, nil)
		db.EXPECT().RemoveTrigger("trigger-1").Return(nil)

		ttl := int64(2 * time.Hour.Seconds())
		err := handleRemoveUnusedTriggersWithTTL(logger, db, ttl)
		So(err, ShouldBeNil)
	})

	Convey("Error delete triggers: error while getting unused triggers, has error", t, func() {
		db.EXPECT().GetUnusedTriggerIDs().Return([]string{"trigger-1"}, errors.New("error"))

		ttl := int64(2 * time.Hour.Seconds())
		err := handleRemoveUnusedTriggersWithTTL(logger, db, ttl)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldResemble, "can't get unused trigger IDs; err: error")
	})

	Convey("Error delete triggers: error while get one trigger, has no error", t, func() {
		db.EXPECT().GetUnusedTriggerIDs().Return([]string{"trigger-1"}, nil)
		db.EXPECT().GetTrigger("trigger-1").Return(moira.Trigger{}, errors.New("error"))

		ttl := int64(2 * time.Hour.Seconds())
		err := handleRemoveUnusedTriggersWithTTL(logger, db, ttl)
		So(err, ShouldBeNil)
	})

	Convey("Error delete triggers: error while delete one trigger, has error", t, func() {
		db.EXPECT().GetUnusedTriggerIDs().Return([]string{"trigger-1"}, nil)
		db.EXPECT().GetTrigger("trigger-1").Return(moira.Trigger{}, nil)
		db.EXPECT().RemoveTrigger("trigger-1").Return(errors.New("error"))

		ttl := int64(2 * time.Hour.Seconds())
		err := handleRemoveUnusedTriggersWithTTL(logger, db, ttl)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldResemble, "can't remove trigger with id trigger-1: error")
	})
}
