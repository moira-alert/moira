package controller

import (
	"fmt"
	"testing"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/internal/api"
	"github.com/moira-alert/moira/internal/api/dto"
	mock_moira_alert "github.com/moira-alert/moira/internal/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetEvents(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()
	triggerID := uuid.Must(uuid.NewV4()).String()
	var page int64 = 10
	var size int64 = 100

	Convey("Test has events", t, func() {
		var total int64 = 6000000
		dataBase.EXPECT().GetNotificationEvents(triggerID, page*size, size-1).Return([]*moira2.NotificationEvent{{State: moira2.StateNODATA, OldState: moira2.StateOK}, {State: moira2.StateOK, OldState: moira2.StateNODATA}}, nil)
		dataBase.EXPECT().GetNotificationEventCount(triggerID, int64(-1)).Return(total)
		list, err := GetTriggerEvents(dataBase, triggerID, page, size)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.EventsList{
			List:  []moira2.NotificationEvent{{State: moira2.StateNODATA, OldState: moira2.StateOK}, {State: moira2.StateOK, OldState: moira2.StateNODATA}},
			Total: total,
			Size:  size,
			Page:  page,
		})
	})

	Convey("Test no events", t, func() {
		var total int64
		dataBase.EXPECT().GetNotificationEvents(triggerID, page*size, size-1).Return(make([]*moira2.NotificationEvent, 0), nil)
		dataBase.EXPECT().GetNotificationEventCount(triggerID, int64(-1)).Return(total)
		list, err := GetTriggerEvents(dataBase, triggerID, page, size)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.EventsList{
			List:  make([]moira2.NotificationEvent, 0),
			Total: total,
			Size:  size,
			Page:  page,
		})
	})

	Convey("Test error", t, func() {
		expected := fmt.Errorf("oooops! Can not get all contacts")
		dataBase.EXPECT().GetNotificationEvents(triggerID, page*size, size-1).Return(nil, expected)
		list, err := GetTriggerEvents(dataBase, triggerID, page, size)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})
}

func TestDeleteAllNotificationEvents(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	Convey("Success", t, func() {
		dataBase.EXPECT().RemoveAllNotificationEvents().Return(nil)
		err := DeleteAllEvents(dataBase)
		So(err, ShouldBeNil)
	})

	Convey("Error delete", t, func() {
		expected := fmt.Errorf("oooops! Can not get notifications")
		dataBase.EXPECT().RemoveAllNotificationEvents().Return(expected)
		err := DeleteAllEvents(dataBase)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}
