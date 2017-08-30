package controller

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/dto"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	"github.com/satori/go.uuid"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestCreateTrigger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	trigger := moira.Trigger{ID: uuid.NewV4().String()}

	Convey("Success", t, func() {
		database.EXPECT().AcquireTriggerCheckLock(gomock.Any(), 10)
		database.EXPECT().DeleteTriggerCheckLock(gomock.Any())
		database.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(nil, nil)
		database.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any()).Return(nil)
		database.EXPECT().SaveTrigger(gomock.Any(), &trigger).Return(nil)
		resp, err := CreateTrigger(database, &trigger, make(map[string]bool))
		So(err, ShouldBeNil)
		So(resp.Message, ShouldResemble, "trigger created")
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("Soo bad trigger")
		database.EXPECT().AcquireTriggerCheckLock(gomock.Any(), 10)
		database.EXPECT().DeleteTriggerCheckLock(gomock.Any())
		database.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(nil, nil)
		database.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any()).Return(nil)
		database.EXPECT().SaveTrigger(gomock.Any(), &trigger).Return(expected)
		resp, err := CreateTrigger(database, &trigger, make(map[string]bool))
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(resp, ShouldBeNil)
	})
}

func TestGetAllTriggers(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Has triggers", t, func() {
		triggerIDs := []string{uuid.NewV4().String(), uuid.NewV4().String()}
		triggers := []moira.TriggerChecks{{Trigger: moira.Trigger{ID: triggerIDs[0]}}, {Trigger: moira.Trigger{ID: triggerIDs[1]}}}
		database.EXPECT().GetTriggerIDs().Return(triggerIDs, nil)
		database.EXPECT().GetTriggerChecks(triggerIDs).Return(triggers, nil)
		list, err := GetAllTriggers(database)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.TriggersList{List: triggers})
	})

	Convey("No triggers", t, func() {
		database.EXPECT().GetTriggerIDs().Return(make([]string, 0), nil)
		database.EXPECT().GetTriggerChecks(make([]string, 0)).Return(make([]moira.TriggerChecks, 0), nil)
		list, err := GetAllTriggers(database)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.TriggersList{List: make([]moira.TriggerChecks, 0)})
	})

	Convey("GetTriggerIDs error", t, func() {
		expected := fmt.Errorf("GetTriggerIDs error")
		database.EXPECT().GetTriggerIDs().Return(nil, expected)
		list, err := GetAllTriggers(database)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})

	Convey("GetTriggerChecks error", t, func() {
		expected := fmt.Errorf("GetTriggerChecks error")
		database.EXPECT().GetTriggerIDs().Return(make([]string, 0), nil)
		database.EXPECT().GetTriggerChecks(make([]string, 0)).Return(nil, expected)
		list, err := GetAllTriggers(database)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})
}

func TestGetTriggerIdsRange(t *testing.T) {
	triggers := make([]string, 20)
	for i := range triggers {
		triggers[i] = uuid.NewV4().String()
	}
	Convey("Has triggers in range", t, func() {
		expected := getTriggerIdsRange(triggers, 20, 3, 5)
		So(expected, ShouldResemble, triggers[15:20])

		expected = getTriggerIdsRange(triggers, 20, 2, 5)
		So(expected, ShouldResemble, triggers[10:15])
	})

	Convey("No triggers on range", t, func() {
		expected := getTriggerIdsRange(triggers, 20, 4, 5)
		So(expected, ShouldResemble, make([]string, 0))

		expected = getTriggerIdsRange(triggers, 20, 55, 1)
		So(expected, ShouldResemble, make([]string, 0))

		expected = getTriggerIdsRange(triggers, 20, 3, 10)
		So(expected, ShouldResemble, make([]string, 0))
	})

	Convey("Range takes part or triggers", t, func() {
		expected := getTriggerIdsRange(triggers, 20, 3, 6)
		So(expected, ShouldResemble, triggers[18:20])

		expected = getTriggerIdsRange(triggers, 20, 1, 11)
		So(expected, ShouldResemble, triggers[11:20])

		expected = getTriggerIdsRange(triggers, 20, 0, 30)
		So(expected, ShouldResemble, triggers)
	})
}

func TestGetNotFilteredTriggers(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggers := make([]string, 20)
	for i := range triggers {
		triggers[i] = uuid.NewV4().String()
	}

	Convey("Get all triggers", t, func() {
		var exp int64 = 20
		database.EXPECT().GetTriggerCheckIDs().Return(triggers, exp, nil)
		database.EXPECT().GetTriggerChecks(triggers).Return(make([]moira.TriggerChecks, 20), nil)
		triggersChecks, total, err := getNotFilteredTriggers(database, 0, 30)
		So(err, ShouldBeNil)
		So(total, ShouldEqual, exp)
		So(triggersChecks, ShouldResemble, make([]moira.TriggerChecks, 20))
	})

	Convey("Get only page", t, func() {
		var exp int64 = 20
		database.EXPECT().GetTriggerCheckIDs().Return(triggers, exp, nil)
		database.EXPECT().GetTriggerChecks(triggers[0:10]).Return(make([]moira.TriggerChecks, 10), nil)
		triggersChecks, total, err := getNotFilteredTriggers(database, 0, 10)
		So(err, ShouldBeNil)
		So(total, ShouldEqual, exp)
		So(triggersChecks, ShouldResemble, make([]moira.TriggerChecks, 10))
	})

	Convey("GetTriggerCheckIDs error", t, func() {
		var exp int64
		expected := fmt.Errorf("GetTriggerCheckIDs error")
		database.EXPECT().GetTriggerCheckIDs().Return(nil, exp, expected)
		triggersChecks, total, err := getNotFilteredTriggers(database, 0, 20)
		So(err, ShouldResemble, expected)
		So(total, ShouldEqual, exp)
		So(triggersChecks, ShouldBeNil)
	})

	Convey("GetTriggerChecks error", t, func() {
		var exp int64 = 20
		expected := fmt.Errorf("GetTriggerChecks error")
		database.EXPECT().GetTriggerCheckIDs().Return(triggers, exp, nil)
		database.EXPECT().GetTriggerChecks(triggers[0:10]).Return(nil, expected)
		triggersChecks, total, err := getNotFilteredTriggers(database, 0, 10)
		So(err, ShouldResemble, expected)
		So(total, ShouldEqual, 0)
		So(triggersChecks, ShouldBeNil)
	})
}

func TestGetFilteredTriggers(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggers := make([]string, 20)
	for i := range triggers {
		triggers[i] = uuid.NewV4().String()
	}
	tags := []string{"tag1", "tag2"}

	Convey("Get all triggers", t, func() {
		var exp int64 = 20
		database.EXPECT().GetFilteredTriggerCheckIds(tags, false).Return(triggers, exp, nil)
		database.EXPECT().GetTriggerChecks(triggers).Return(make([]moira.TriggerChecks, 20), nil)
		triggersChecks, total, err := getFilteredTriggers(database, 0, 30, false, tags)
		So(err, ShouldBeNil)
		So(total, ShouldEqual, exp)
		So(triggersChecks, ShouldResemble, make([]moira.TriggerChecks, 20))
	})

	Convey("Get only page", t, func() {
		var exp int64 = 20
		database.EXPECT().GetFilteredTriggerCheckIds(tags, false).Return(triggers, exp, nil)
		database.EXPECT().GetTriggerChecks(triggers[0:10]).Return(make([]moira.TriggerChecks, 10), nil)
		triggersChecks, total, err := getFilteredTriggers(database, 0, 10, false, tags)
		So(err, ShouldBeNil)
		So(total, ShouldEqual, exp)
		So(triggersChecks, ShouldResemble, make([]moira.TriggerChecks, 10))
	})

	Convey("GetFilteredTriggerCheckIds error", t, func() {
		var exp int64
		expected := fmt.Errorf("GetFilteredTriggerCheckIds error")
		database.EXPECT().GetFilteredTriggerCheckIds(tags, true).Return(nil, exp, expected)
		triggersChecks, total, err := getFilteredTriggers(database, 0, 20, true, tags)
		So(err, ShouldResemble, expected)
		So(total, ShouldEqual, exp)
		So(triggersChecks, ShouldBeNil)
	})

	Convey("GetTriggerChecks error", t, func() {
		var exp int64 = 20
		expected := fmt.Errorf("GetTriggerChecks error")
		database.EXPECT().GetFilteredTriggerCheckIds(tags, true).Return(triggers, exp, nil)
		database.EXPECT().GetTriggerChecks(triggers[0:10]).Return(nil, expected)
		triggersChecks, total, err := getFilteredTriggers(database, 0, 10, true, tags)
		So(err, ShouldResemble, expected)
		So(total, ShouldEqual, 0)
		So(triggersChecks, ShouldBeNil)
	})
}

func TestGetTriggerPage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	var page int64
	var size int64 = 10
	triggers := make([]string, 20)
	for i := range triggers {
		triggers[i] = uuid.NewV4().String()
	}

	Convey("Has tags and only errors", t, func() {
		tags := []string{"tag1", "tag2"}
		var exp int64 = 20
		database.EXPECT().GetFilteredTriggerCheckIds(tags, true).Return(triggers, exp, nil)
		database.EXPECT().GetTriggerChecks(triggers[0:10]).Return(make([]moira.TriggerChecks, 10), nil)
		list, err := GetTriggerPage(database, page, size, true, tags)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.TriggersList{
			List:  make([]moira.TriggerChecks, 10),
			Total: &exp,
			Page:  &page,
			Size:  &size,
		})
	})

	Convey("All triggers", t, func() {
		var exp int64 = 20
		database.EXPECT().GetTriggerCheckIDs().Return(triggers, exp, nil)
		database.EXPECT().GetTriggerChecks(triggers[0:10]).Return(make([]moira.TriggerChecks, 10), nil)
		list, err := GetTriggerPage(database, page, size, false, make([]string, 0))
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.TriggersList{
			List:  make([]moira.TriggerChecks, 10),
			Total: &exp,
			Page:  &page,
			Size:  &size,
		})
	})

	Convey("No tags only errors and error GetFilteredTriggerCheckIds", t, func() {
		var exp int64
		expected := fmt.Errorf("GetFilteredTriggerCheckIds error")
		database.EXPECT().GetFilteredTriggerCheckIds(make([]string, 0), true).Return(nil, exp, expected)
		list, err := GetTriggerPage(database, 0, 20, true, make([]string, 0))
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})
}
