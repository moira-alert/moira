package controller

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/satori/go.uuid"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCreateTrigger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Success with trigger.ID empty", t, func() {
		triggerModel := dto.TriggerModel{}
		dataBase.EXPECT().AcquireTriggerCheckLock(gomock.Any(), 10)
		dataBase.EXPECT().DeleteTriggerCheckLock(gomock.Any())
		dataBase.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(moira.CheckData{}, database.ErrNil)
		dataBase.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		dataBase.EXPECT().SaveTrigger(gomock.Any(), gomock.Any()).Return(nil)
		resp, err := CreateTrigger(dataBase, &triggerModel, make(map[string]bool))
		So(err, ShouldBeNil)
		So(resp.Message, ShouldResemble, "trigger created")
	})

	Convey("Success with triggerID", t, func() {
		triggerID := uuid.NewV4().String()
		triggerModel := dto.TriggerModel{ID: triggerID}
		dataBase.EXPECT().GetTrigger(triggerModel.ID).Return(moira.Trigger{}, database.ErrNil)
		dataBase.EXPECT().AcquireTriggerCheckLock(gomock.Any(), 10)
		dataBase.EXPECT().DeleteTriggerCheckLock(gomock.Any())
		dataBase.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(moira.CheckData{}, database.ErrNil)
		dataBase.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		dataBase.EXPECT().SaveTrigger(gomock.Any(), triggerModel.ToMoiraTrigger()).Return(nil)
		resp, err := CreateTrigger(dataBase, &triggerModel, make(map[string]bool))
		So(err, ShouldBeNil)
		So(resp.Message, ShouldResemble, "trigger created")
		So(resp.ID, ShouldResemble, triggerID)
	})

	Convey("Trigger already exists", t, func() {
		triggerModel := dto.TriggerModel{ID: uuid.NewV4().String()}
		trigger := triggerModel.ToMoiraTrigger()
		dataBase.EXPECT().GetTrigger(triggerModel.ID).Return(*trigger, nil)
		resp, err := CreateTrigger(dataBase, &triggerModel, make(map[string]bool))
		So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("trigger with this ID already exists")))
		So(resp, ShouldBeNil)
	})

	Convey("Get trigger error", t, func() {
		trigger := dto.TriggerModel{ID: uuid.NewV4().String()}
		expected := fmt.Errorf("soo bad trigger")
		dataBase.EXPECT().GetTrigger(trigger.ID).Return(moira.Trigger{}, expected)
		resp, err := CreateTrigger(dataBase, &trigger, make(map[string]bool))
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(resp, ShouldBeNil)
	})

	Convey("Error", t, func() {
		triggerModel := dto.TriggerModel{ID: uuid.NewV4().String()}
		expected := fmt.Errorf("soo bad trigger")
		dataBase.EXPECT().GetTrigger(triggerModel.ID).Return(moira.Trigger{}, database.ErrNil)
		dataBase.EXPECT().AcquireTriggerCheckLock(gomock.Any(), 10)
		dataBase.EXPECT().DeleteTriggerCheckLock(gomock.Any())
		dataBase.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(moira.CheckData{}, database.ErrNil)
		dataBase.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		dataBase.EXPECT().SaveTrigger(gomock.Any(), triggerModel.ToMoiraTrigger()).Return(expected)
		resp, err := CreateTrigger(dataBase, &triggerModel, make(map[string]bool))
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(resp, ShouldBeNil)
	})
}

func TestGetAllTriggers(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Has triggers", t, func() {
		triggerIDs := []string{uuid.NewV4().String(), uuid.NewV4().String()}
		triggers := []*moira.TriggerCheck{{Trigger: moira.Trigger{ID: triggerIDs[0]}}, {Trigger: moira.Trigger{ID: triggerIDs[1]}}}
		triggersList := []moira.TriggerCheck{{Trigger: moira.Trigger{ID: triggerIDs[0]}}, {Trigger: moira.Trigger{ID: triggerIDs[1]}}}
		mockDatabase.EXPECT().GetAllTriggerIDs().Return(triggerIDs, nil)
		mockDatabase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggers, nil)
		list, err := GetAllTriggers(mockDatabase)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.TriggersList{List: triggersList})
	})

	Convey("No triggers", t, func() {
		mockDatabase.EXPECT().GetAllTriggerIDs().Return(make([]string, 0), nil)
		mockDatabase.EXPECT().GetTriggerChecks(make([]string, 0)).Return(make([]*moira.TriggerCheck, 0), nil)
		list, err := GetAllTriggers(mockDatabase)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.TriggersList{List: make([]moira.TriggerCheck, 0)})
	})

	Convey("GetTriggerIDs error", t, func() {
		expected := fmt.Errorf("getTriggerIDs error")
		mockDatabase.EXPECT().GetAllTriggerIDs().Return(nil, expected)
		list, err := GetAllTriggers(mockDatabase)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})

	Convey("GetTriggerChecks error", t, func() {
		expected := fmt.Errorf("getTriggerChecks error")
		mockDatabase.EXPECT().GetAllTriggerIDs().Return(make([]string, 0), nil)
		mockDatabase.EXPECT().GetTriggerChecks(make([]string, 0)).Return(nil, expected)
		list, err := GetAllTriggers(mockDatabase)
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

func TestGetTriggerPage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
	var page int64
	var size int64 = 10
	triggerIDs := make([]string, 20)
	for i := range triggerIDs {
		triggerIDs[i] = uuid.NewV4().String()
	}
	triggers := make([]moira.TriggerCheck, 20)
	for i := range triggerIDs {
		triggers[i] = moira.TriggerCheck{Trigger: moira.Trigger{ID: triggerIDs[i]}}
	}
	triggersPointers := make([]*moira.TriggerCheck, 20)
	for i := range triggerIDs {
		triggersPointers[i] = &moira.TriggerCheck{Trigger: moira.Trigger{ID: triggerIDs[i]}}
	}

	Convey("Has tags and only errors", t, func() {
		tags := []string{"tag1", "tag2"}
		var exp int64 = 20
		mockDatabase.EXPECT().GetTriggerCheckIDs(tags, true).Return(triggerIDs, nil)
		mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[0:10]).Return(triggersPointers[0:10], nil)
		list, err := GetTriggerPage(mockDatabase, page, size, true, tags)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.TriggersList{
			List:  triggers[0:10],
			Total: &exp,
			Page:  &page,
			Size:  &size,
		})
	})

	Convey("All triggers", t, func() {
		var exp int64 = 20
		mockDatabase.EXPECT().GetTriggerCheckIDs(make([]string, 0), false).Return(triggerIDs, nil)
		mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[0:10]).Return(triggersPointers[0:10], nil)
		list, err := GetTriggerPage(mockDatabase, page, size, false, make([]string, 0))
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.TriggersList{
			List:  triggers[0:10],
			Total: &exp,
			Page:  &page,
			Size:  &size,
		})
	})

	Convey("Error GetFilteredTriggerCheckIDs", t, func() {
		expected := fmt.Errorf("getFilteredTriggerCheckIDs error")
		mockDatabase.EXPECT().GetTriggerCheckIDs(make([]string, 0), true).Return(nil, expected)
		list, err := GetTriggerPage(mockDatabase, 0, 20, true, make([]string, 0))
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})

	Convey("Error GetTriggerChecks", t, func() {
		expected := fmt.Errorf("getTriggerChecks error")
		mockDatabase.EXPECT().GetTriggerCheckIDs(make([]string, 0), false).Return(triggerIDs, nil)
		mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[0:10]).Return(nil, expected)
		list, err := GetTriggerPage(mockDatabase, page, size, false, make([]string, 0))
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})
}
