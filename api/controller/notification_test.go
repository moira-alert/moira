package controller

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/dto"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestGetNotifications(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	var start int64 = 100
	var end int64 = 33

	Convey("Has notifications", t, func() {
		notifications := []*moira.ScheduledNotification{{Timestamp: 123, SendFail: 6}, {Timestamp: 321, SendFail: 1}}
		var total int64 = 666
		dataBase.EXPECT().GetNotifications(start, end).Return(notifications, total, nil)
		list, err := GetNotifications(dataBase, start, end)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.NotificationsList{List: notifications, Total: total})
	})

	Convey("Test error", t, func() {
		expected := fmt.Errorf("Oooops! Can not get notifications")
		var total int64 = 666
		dataBase.EXPECT().GetNotifications(start, end).Return(nil, total, expected)
		list, err := GetNotifications(dataBase, start, end)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})
}

func TestDeleteNotification(t *testing.T) {
	mockCtrl := gomock.NewController(t)
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
		var result int64 = 0
		expected := fmt.Errorf("Oooops! Can not get notifications")
		dataBase.EXPECT().RemoveNotification(key).Return(result, expected)
		actual, err := DeleteNotification(dataBase, key)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(actual, ShouldBeNil)
	})
}
