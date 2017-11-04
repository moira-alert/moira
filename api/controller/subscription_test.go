package controller

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/mock/moira-alert"
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
		subscriptions := []*moira.SubscriptionData{{ID: subscriptionIDs[0]}, {ID: subscriptionIDs[1]}}
		database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
		database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
		list, err := GetUserSubscriptions(database, login)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.SubscriptionList{List: []moira.SubscriptionData{*subscriptions[0], *subscriptions[1]}})
	})

	Convey("Two ids, one subscription", t, func() {
		subscriptionIDs := []string{uuid.NewV4().String(), uuid.NewV4().String()}
		subscriptions := []*moira.SubscriptionData{{ID: subscriptionIDs[1]}}
		database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
		database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
		list, err := GetUserSubscriptions(database, login)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.SubscriptionList{List: []moira.SubscriptionData{*subscriptions[0]}})
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

func TestUpdateSubscription(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()
	userLogin := "user"

	Convey("Success update", t, func() {
		subscriptionDTO := &dto.Subscription{}
		subscriptionID := uuid.NewV4().String()
		subscription := moira.SubscriptionData{
			ID:   subscriptionID,
			User: userLogin,
		}
		dataBase.EXPECT().SaveSubscription(&subscription).Return(nil)
		err := UpdateSubscription(dataBase, subscriptionID, userLogin, subscriptionDTO)
		So(err, ShouldBeNil)
		So(subscriptionDTO.User, ShouldResemble, userLogin)
		So(subscriptionDTO.ID, ShouldResemble, subscriptionID)
	})

	Convey("Error save", t, func() {
		subscriptionDTO := &dto.Subscription{}
		subscriptionID := uuid.NewV4().String()
		subscription := moira.SubscriptionData{
			ID:   subscriptionID,
			User: userLogin,
		}
		err := fmt.Errorf("Oooops")
		dataBase.EXPECT().SaveSubscription(&subscription).Return(err)
		actual := UpdateSubscription(dataBase, subscriptionID, userLogin, subscriptionDTO)
		So(actual, ShouldResemble, api.ErrorInternalServer(err))
		So(subscriptionDTO.User, ShouldResemble, userLogin)
		So(subscriptionDTO.ID, ShouldResemble, subscriptionID)
	})
}

func TestRemoveSubscription(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	id := uuid.NewV4().String()

	Convey("Success", t, func() {
		database.EXPECT().RemoveSubscription(id).Return(nil)
		err := RemoveSubscription(database, id)
		So(err, ShouldBeNil)
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("Oooops! Can not remove subscription")
		database.EXPECT().RemoveSubscription(id).Return(expected)
		err := RemoveSubscription(database, id)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestSendTestNotification(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	id := uuid.NewV4().String()

	Convey("Success", t, func() {
		database.EXPECT().PushNotificationEvent(gomock.Any(), false).Return(nil)
		err := SendTestNotification(database, id)
		So(err, ShouldBeNil)
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("Oooops! Can not push event")
		database.EXPECT().PushNotificationEvent(gomock.Any(), false).Return(expected)
		err := SendTestNotification(database, id)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestCreateSubscription(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	login := "user"

	Convey("Success create", t, func() {
		subscription := dto.Subscription{ID: ""}
		dataBase.EXPECT().SaveSubscription(gomock.Any()).Return(nil)
		err := CreateSubscription(dataBase, login, &subscription)
		So(err, ShouldBeNil)
	})

	Convey("Success create subscription with id", t, func() {
		sub := &dto.Subscription{
			ID: uuid.NewV4().String(),
		}
		dataBase.EXPECT().GetSubscription(sub.ID).Return(moira.SubscriptionData{}, database.ErrNil)
		dataBase.EXPECT().SaveSubscription(gomock.Any()).Return(nil)
		err := CreateSubscription(dataBase, login, sub)
		So(err, ShouldBeNil)
		So(sub.User, ShouldResemble, login)
		So(sub.ID, ShouldResemble, sub.ID)
	})

	Convey("Subscription exists by id", t, func() {
		subscription := &dto.Subscription{
			ID: uuid.NewV4().String(),
		}
		dataBase.EXPECT().GetSubscription(subscription.ID).Return(moira.SubscriptionData{}, nil)
		err := CreateSubscription(dataBase, login, subscription)
		So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("Subscription with this ID already exists")))
	})

	Convey("Error get subscription", t, func() {
		subscription := &dto.Subscription{
			ID: uuid.NewV4().String(),
		}
		err := fmt.Errorf("Oooops! Can not write contact")
		dataBase.EXPECT().GetSubscription(subscription.ID).Return(moira.SubscriptionData{}, err)
		expected := CreateSubscription(dataBase, login, subscription)
		So(expected, ShouldResemble, api.ErrorInternalServer(err))
	})

	Convey("Error save subscription", t, func() {
		subscription := dto.Subscription{ID: ""}
		expected := fmt.Errorf("Oooops! Can not create subscription")
		dataBase.EXPECT().SaveSubscription(gomock.Any()).Return(expected)
		err := CreateSubscription(dataBase, login, &subscription)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestCheckUserPermissionsForSubscription(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	userLogin := uuid.NewV4().String()
	id := uuid.NewV4().String()

	Convey("No subscription", t, func() {
		dataBase.EXPECT().GetSubscription(id).Return(moira.SubscriptionData{}, database.ErrNil)
		expected := CheckUserPermissionsForSubscription(dataBase, id, userLogin)
		So(expected, ShouldResemble, api.ErrorNotFound(fmt.Sprintf("Subscription with ID '%s' does not exists", id)))
	})

	Convey("Different user", t, func() {
		dataBase.EXPECT().GetSubscription(id).Return(moira.SubscriptionData{User: "diffUser"}, nil)
		expected := CheckUserPermissionsForSubscription(dataBase, id, userLogin)
		So(expected, ShouldResemble, api.ErrorForbidden("You have not permissions"))
	})

	Convey("Has subscription", t, func() {
		dataBase.EXPECT().GetSubscription(id).Return(moira.SubscriptionData{User: userLogin}, nil)
		expected := CheckUserPermissionsForSubscription(dataBase, id, userLogin)
		So(expected, ShouldBeNil)
	})

	Convey("Error get contact", t, func() {
		err := fmt.Errorf("Oooops! Can not read contact")
		dataBase.EXPECT().GetSubscription(id).Return(moira.SubscriptionData{User: userLogin}, err)
		expected := CheckUserPermissionsForSubscription(dataBase, id, userLogin)
		So(expected, ShouldResemble, api.ErrorInternalServer(err))
	})
}
