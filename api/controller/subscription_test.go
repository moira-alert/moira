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

func TestGetUserSubscriptions(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	login := "user"

	Convey("Two subscriptions", t, func() {
		subscriptionIDs := []string{uuid.NewV4().String(), uuid.NewV4().String()}
		subscriptions := []moira.SubscriptionData{{ID: subscriptionIDs[0]}, {ID: subscriptionIDs[1]}}
		database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
		database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
		list, err := GetUserSubscriptions(database, login)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.SubscriptionList{List: subscriptions})
	})

	Convey("Two ids, one subscription", t, func() {
		subscriptionIDs := []string{uuid.NewV4().String(), uuid.NewV4().String()}
		subscriptions := []moira.SubscriptionData{{ID: subscriptionIDs[1]}}
		database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
		database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
		list, err := GetUserSubscriptions(database, login)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.SubscriptionList{List: subscriptions})
	})

	Convey("Errors", t, func() {
		Convey("GetUserSubscriptionIDs", func() {
			expected := fmt.Errorf("Oh no!!!11 Cant get subscription ids")
			database.EXPECT().GetUserSubscriptionIDs(login).Return(nil, expected)
			list, err := GetUserSubscriptions(database, login)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(list, ShouldBeNil)
		})

		Convey("GetSubscriptions", func() {
			expected := fmt.Errorf("Oh no!!!11 Cant get subscriptions")
			subscriptionIDs := []string{uuid.NewV4().String(), uuid.NewV4().String()}
			database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
			database.EXPECT().GetSubscriptions(subscriptionIDs).Return(nil, expected)
			list, err := GetUserSubscriptions(database, login)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(list, ShouldBeNil)
		})
	})
}

func TestDeleteSubscription(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	login := "user"
	id := uuid.NewV4().String()

	Convey("Success", t, func() {
		database.EXPECT().DeleteSubscription(id, login).Return(nil)
		err := DeleteSubscription(database, id, login)
		So(err, ShouldBeNil)
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("Oooops! Can not remove subscription")
		database.EXPECT().DeleteSubscription(id, login).Return(expected)
		err := DeleteSubscription(database, id, login)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestSendTestNotification(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	id := uuid.NewV4().String()

	Convey("Success", t, func() {
		database.EXPECT().PushEvent(gomock.Any(), false).Return(nil)
		err := SendTestNotification(database, id)
		So(err, ShouldBeNil)
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("Oooops! Can not push event")
		database.EXPECT().PushEvent(gomock.Any(), false).Return(expected)
		err := SendTestNotification(database, id)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestWriteSubscription(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	login := "user"

	Convey("Create subscription", t, func() {
		subscription := dto.Subscription{ID: ""}
		Convey("Success", func() {
			database.EXPECT().CreateSubscription(gomock.Any()).Return(nil)
			err := WriteSubscription(database, login, &subscription)
			So(err, ShouldBeNil)
		})

		Convey("Error", func() {
			expected := fmt.Errorf("Oooops! Can not create subscription")
			database.EXPECT().CreateSubscription(gomock.Any()).Return(expected)
			err := WriteSubscription(database, login, &subscription)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
		})
	})

	Convey("Update subscription", t, func() {
		subscription := dto.Subscription{ID: uuid.NewV4().String()}
		Convey("Success", func() {
			database.EXPECT().UpdateSubscription(gomock.Any()).Return(nil)
			err := WriteSubscription(database, login, &subscription)
			So(err, ShouldBeNil)
		})

		Convey("Error", func() {
			expected := fmt.Errorf("Oooops! Can not update subscription")
			database.EXPECT().UpdateSubscription(gomock.Any()).Return(expected)
			err := WriteSubscription(database, login, &subscription)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
		})
	})

}
