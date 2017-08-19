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

func TestGetUserSettings(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	login := "user"

	Convey("Success get user data", t, func() {
		subscriptionIDs := []string{uuid.NewV4().String(), uuid.NewV4().String()}
		subscriptions := []moira.SubscriptionData{{ID: subscriptionIDs[0]}, {ID: subscriptionIDs[1]}}
		contactIDs := []string{uuid.NewV4().String(), uuid.NewV4().String()}
		contacts := []moira.ContactData{{ID: contactIDs[0]}, {ID: contactIDs[1]}}
		database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
		database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
		database.EXPECT().GetUserContacts(login).Return(contactIDs, nil)
		database.EXPECT().GetContact(contactIDs[0]).Return(contacts[0], nil)
		database.EXPECT().GetContact(contactIDs[1]).Return(contacts[1], nil)
		settings, err := GetUserSettings(database, login)
		So(err, ShouldBeNil)
		So(settings, ShouldResemble, &dto.UserSettings{
			User:          dto.User{Login: login},
			Contacts:      contacts,
			Subscriptions: subscriptions,
		})
	})

	Convey("No contacts and subscriptions", t, func() {
		database.EXPECT().GetUserSubscriptionIDs(login).Return(make([]string, 0), nil)
		database.EXPECT().GetSubscriptions(make([]string, 0)).Return(make([]moira.SubscriptionData, 0), nil)
		database.EXPECT().GetUserContacts(login).Return(make([]string, 0), nil)
		settings, err := GetUserSettings(database, login)
		So(err, ShouldBeNil)
		So(settings, ShouldResemble, &dto.UserSettings{
			User:          dto.User{Login: login},
			Contacts:      make([]moira.ContactData, 0),
			Subscriptions: make([]moira.SubscriptionData, 0),
		})
	})

	Convey("Errors", t, func() {
		Convey("GetUserSubscriptionIDs", func() {
			expected := fmt.Errorf("Can not read ids")
			database.EXPECT().GetUserSubscriptionIDs(login).Return(nil, expected)
			settings, err := GetUserSettings(database, login)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldBeNil)
		})
		Convey("GetSubscriptions", func() {
			expected := fmt.Errorf("Can not read subscriptions")
			database.EXPECT().GetUserSubscriptionIDs(login).Return(make([]string, 0), nil)
			database.EXPECT().GetSubscriptions(make([]string, 0)).Return(nil, expected)
			settings, err := GetUserSettings(database, login)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldBeNil)
		})
		Convey("GetUserContacts", func() {
			expected := fmt.Errorf("Can not read contact ids")
			database.EXPECT().GetUserSubscriptionIDs(login).Return(make([]string, 0), nil)
			database.EXPECT().GetSubscriptions(make([]string, 0)).Return(make([]moira.SubscriptionData, 0), nil)
			database.EXPECT().GetUserContacts(login).Return(nil, expected)
			settings, err := GetUserSettings(database, login)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldBeNil)
		})
		Convey("GetContact", func() {
			expected := fmt.Errorf("Can not read contact")
			subscriptionIDs := []string{uuid.NewV4().String(), uuid.NewV4().String()}
			subscriptions := []moira.SubscriptionData{{ID: subscriptionIDs[0]}, {ID: subscriptionIDs[1]}}
			contactIDs := []string{uuid.NewV4().String(), uuid.NewV4().String()}
			database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
			database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
			database.EXPECT().GetUserContacts(login).Return(contactIDs, nil)
			database.EXPECT().GetContact(contactIDs[0]).Return(moira.ContactData{}, expected)
			settings, err := GetUserSettings(database, login)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldBeNil)
		})
	})
}
