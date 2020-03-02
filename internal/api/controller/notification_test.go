package controller

import (
	"fmt"
	"testing"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/internal/api"
	"github.com/moira-alert/moira/internal/api/dto"
	mock_moira_alert "github.com/moira-alert/moira/internal/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetNotifications(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	var start int64 = 100
	var end int64 = 33

	Convey("Has notifications", t, func() {
		notifications := []*moira2.ScheduledNotification{{Timestamp: 123, SendFail: 6}, {Timestamp: 321, SendFail: 1}}
		var total int64 = 666
		dataBase.EXPECT().GetNotifications(start, end).Return(notifications, total, nil)
		list, err := GetNotifications(dataBase, start, end)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.NotificationsList{List: notifications, Total: total})
	})

	Convey("Test error", t, func() {
		expected := fmt.Errorf("oooops! Can not get notifications")
		var total int64 = 666
		dataBase.EXPECT().GetNotifications(start, end).Return(nil, total, expected)
		list, err := GetNotifications(dataBase, start, end)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})
}

func TestDeleteNotification(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Success", t, func() {
		key := "123"
		var result int64 = 1
		dataBase.EXPECT().RemoveNotification(key).Return(result, nil)
		actual, err := DeleteNotification(dataBase, key)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.NotificationDeleteResponse{Result: 1})
	})

	Convey("Error delete", t, func() {
		key := "123"
		var result int64
		expected := fmt.Errorf("oooops! Can not get notifications")
		dataBase.EXPECT().RemoveNotification(key).Return(result, expected)
		actual, err := DeleteNotification(dataBase, key)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(actual, ShouldBeNil)
	})
}

func TestDeleteAllNotifications(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Success", t, func() {
		dataBase.EXPECT().RemoveAllNotifications().Return(nil)
		err := DeleteAllNotifications(dataBase)
		So(err, ShouldBeNil)
	})

	Convey("Error delete", t, func() {
		expected := fmt.Errorf("oooops! Can not get notifications")
		dataBase.EXPECT().RemoveAllNotifications().Return(expected)
		err := DeleteAllNotifications(dataBase)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}
