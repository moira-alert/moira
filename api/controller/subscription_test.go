package controller

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetUserSubscriptions(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	login := "user"

	Convey("Two subscriptions", t, func(c C) {
		subscriptionIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
		subscriptions := []*moira.SubscriptionData{{ID: subscriptionIDs[0]}, {ID: subscriptionIDs[1]}}
		database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
		database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
		list, err := GetUserSubscriptions(database, login)
		c.So(err, ShouldBeNil)
		c.So(list, ShouldResemble, &dto.SubscriptionList{List: []moira.SubscriptionData{*subscriptions[0], *subscriptions[1]}})
	})

	Convey("Two ids, one subscription", t, func(c C) {
		subscriptionIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
		subscriptions := []*moira.SubscriptionData{{ID: subscriptionIDs[1]}}
		database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
		database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
		list, err := GetUserSubscriptions(database, login)
		c.So(err, ShouldBeNil)
		c.So(list, ShouldResemble, &dto.SubscriptionList{List: []moira.SubscriptionData{*subscriptions[0]}})
	})

	Convey("Errors", t, func(c C) {
		Convey("GetUserSubscriptionIDs", t, func(c C) {
			expected := fmt.Errorf("oh no!!!11 Cant get subscription ids")
			database.EXPECT().GetUserSubscriptionIDs(login).Return(nil, expected)
			list, err := GetUserSubscriptions(database, login)
			c.So(err, ShouldResemble, api.ErrorInternalServer(expected))
			c.So(list, ShouldBeNil)
		})

		Convey("GetSubscriptions", t, func(c C) {
			expected := fmt.Errorf("oh no!!!11 Cant get subscriptions")
			subscriptionIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
			database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
			database.EXPECT().GetSubscriptions(subscriptionIDs).Return(nil, expected)
			list, err := GetUserSubscriptions(database, login)
			c.So(err, ShouldResemble, api.ErrorInternalServer(expected))
			c.So(list, ShouldBeNil)
		})
	})
}

func TestUpdateSubscription(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()
	userLogin := "user"

	Convey("Success update", t, func(c C) {
		subscriptionDTO := &dto.Subscription{}
		subscriptionID := uuid.Must(uuid.NewV4()).String()
		subscription := moira.SubscriptionData{
			ID:   subscriptionID,
			User: userLogin,
		}
		dataBase.EXPECT().SaveSubscription(&subscription).Return(nil)
		err := UpdateSubscription(dataBase, subscriptionID, userLogin, subscriptionDTO)
		c.So(err, ShouldBeNil)
		c.So(subscriptionDTO.User, ShouldResemble, userLogin)
		c.So(subscriptionDTO.ID, ShouldResemble, subscriptionID)
	})

	Convey("Error save", t, func(c C) {
		subscriptionDTO := &dto.Subscription{}
		subscriptionID := uuid.Must(uuid.NewV4()).String()
		subscription := moira.SubscriptionData{
			ID:   subscriptionID,
			User: userLogin,
		}
		err := fmt.Errorf("oooops")
		dataBase.EXPECT().SaveSubscription(&subscription).Return(err)
		actual := UpdateSubscription(dataBase, subscriptionID, userLogin, subscriptionDTO)
		c.So(actual, ShouldResemble, api.ErrorInternalServer(err))
		c.So(subscriptionDTO.User, ShouldResemble, userLogin)
		c.So(subscriptionDTO.ID, ShouldResemble, subscriptionID)
	})
}

func TestRemoveSubscription(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	db := mock_moira_alert.NewMockDatabase(mockCtrl)
	id := uuid.Must(uuid.NewV4()).String()

	Convey("Success", t, func(c C) {
		db.EXPECT().RemoveSubscription(id).Return(nil)
		err := RemoveSubscription(db, id)
		c.So(err, ShouldBeNil)
	})

	Convey("Error", t, func(c C) {
		expected := fmt.Errorf("oooops! Can not remove subscription")
		db.EXPECT().RemoveSubscription(id).Return(expected)
		err := RemoveSubscription(db, id)
		c.So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestSendTestNotification(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	db := mock_moira_alert.NewMockDatabase(mockCtrl)
	id := uuid.Must(uuid.NewV4()).String()

	Convey("Success", t, func(c C) {
		db.EXPECT().PushNotificationEvent(gomock.Any(), false).Return(nil)
		err := SendTestNotification(db, id)
		c.So(err, ShouldBeNil)
	})

	Convey("Error", t, func(c C) {
		expected := fmt.Errorf("oooops! Can not push event")
		db.EXPECT().PushNotificationEvent(gomock.Any(), false).Return(expected)
		err := SendTestNotification(db, id)
		c.So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestCreateSubscription(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	login := "user"

	Convey("Success create", t, func(c C) {
		subscription := dto.Subscription{ID: ""}
		dataBase.EXPECT().SaveSubscription(gomock.Any()).Return(nil)
		err := CreateSubscription(dataBase, login, &subscription)
		c.So(err, ShouldBeNil)
	})

	Convey("Success create subscription with id", t, func(c C) {
		sub := &dto.Subscription{
			ID: uuid.Must(uuid.NewV4()).String(),
		}
		dataBase.EXPECT().GetSubscription(sub.ID).Return(moira.SubscriptionData{}, database.ErrNil)
		dataBase.EXPECT().SaveSubscription(gomock.Any()).Return(nil)
		err := CreateSubscription(dataBase, login, sub)
		c.So(err, ShouldBeNil)
		c.So(sub.User, ShouldResemble, login)
		c.So(sub.ID, ShouldResemble, sub.ID)
	})

	Convey("Subscription exists by id", t, func(c C) {
		subscription := &dto.Subscription{
			ID: uuid.Must(uuid.NewV4()).String(),
		}
		dataBase.EXPECT().GetSubscription(subscription.ID).Return(moira.SubscriptionData{}, nil)
		err := CreateSubscription(dataBase, login, subscription)
		c.So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("subscription with this ID already exists")))
	})

	Convey("Error get subscription", t, func(c C) {
		subscription := &dto.Subscription{
			ID: uuid.Must(uuid.NewV4()).String(),
		}
		err := fmt.Errorf("oooops! Can not write contact")
		dataBase.EXPECT().GetSubscription(subscription.ID).Return(moira.SubscriptionData{}, err)
		expected := CreateSubscription(dataBase, login, subscription)
		c.So(expected, ShouldResemble, api.ErrorInternalServer(err))
	})

	Convey("Error save subscription", t, func(c C) {
		subscription := dto.Subscription{ID: ""}
		expected := fmt.Errorf("oooops! Can not create subscription")
		dataBase.EXPECT().SaveSubscription(gomock.Any()).Return(expected)
		err := CreateSubscription(dataBase, login, &subscription)
		c.So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestCheckUserPermissionsForSubscription(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	userLogin := uuid.Must(uuid.NewV4()).String()
	id := uuid.Must(uuid.NewV4()).String()

	Convey("No subscription", t, func(c C) {
		dataBase.EXPECT().GetSubscription(id).Return(moira.SubscriptionData{}, database.ErrNil)
		expectedSub, expected := CheckUserPermissionsForSubscription(dataBase, id, userLogin)
		c.So(expected, ShouldResemble, api.ErrorNotFound(fmt.Sprintf("subscription with ID '%s' does not exists", id)))
		c.So(expectedSub, ShouldResemble, moira.SubscriptionData{})
	})

	Convey("Different user", t, func(c C) {
		actualSub := moira.SubscriptionData{User: "diffUser"}
		dataBase.EXPECT().GetSubscription(id).Return(actualSub, nil)
		expectedSub, expected := CheckUserPermissionsForSubscription(dataBase, id, userLogin)
		c.So(expected, ShouldResemble, api.ErrorForbidden("you are not permitted"))
		c.So(expectedSub, ShouldResemble, actualSub)
	})

	Convey("Has subscription", t, func(c C) {
		actualSub := moira.SubscriptionData{ID: id, User: userLogin}
		dataBase.EXPECT().GetSubscription(id).Return(actualSub, nil)
		expectedSub, expected := CheckUserPermissionsForSubscription(dataBase, id, userLogin)
		c.So(expected, ShouldBeNil)
		c.So(expectedSub, ShouldResemble, actualSub)
	})

	Convey("Error get contact", t, func(c C) {
		err := fmt.Errorf("oooops! Can not read contact")
		dataBase.EXPECT().GetSubscription(id).Return(moira.SubscriptionData{}, err)
		expectedSub, expected := CheckUserPermissionsForSubscription(dataBase, id, userLogin)
		c.So(expected, ShouldResemble, api.ErrorInternalServer(err))
		c.So(expectedSub, ShouldResemble, moira.SubscriptionData{})
	})
}
