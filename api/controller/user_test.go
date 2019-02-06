package controller

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetUserSettings(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	login := "user"

	Convey("Success get user data", t, func() {
		subscriptionIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
		subscriptions := []*moira.SubscriptionData{{ID: subscriptionIDs[0]}, {ID: subscriptionIDs[1]}}
		contactIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
		contacts := []*moira.ContactData{{ID: contactIDs[0]}, {ID: contactIDs[1]}}
		database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
		database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
		database.EXPECT().GetUserContactIDs(login).Return(contactIDs, nil)
		database.EXPECT().GetContacts(contactIDs).Return(contacts, nil)
		settings, err := GetUserSettings(database, login)
		So(err, ShouldBeNil)
		So(settings, ShouldResemble, &dto.UserSettings{
			User:          dto.User{Login: login},
			Contacts:      []moira.ContactData{*contacts[0], *contacts[1]},
			Subscriptions: []moira.SubscriptionData{*subscriptions[0], *subscriptions[1]},
		})
	})

	Convey("No contacts and subscriptions", t, func() {
		database.EXPECT().GetUserSubscriptionIDs(login).Return(make([]string, 0), nil)
		database.EXPECT().GetSubscriptions(make([]string, 0)).Return(make([]*moira.SubscriptionData, 0), nil)
		database.EXPECT().GetUserContactIDs(login).Return(make([]string, 0), nil)
		database.EXPECT().GetContacts(make([]string, 0)).Return(make([]*moira.ContactData, 0), nil)
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
			expected := fmt.Errorf("can not read ids")
			database.EXPECT().GetUserSubscriptionIDs(login).Return(nil, expected)
			settings, err := GetUserSettings(database, login)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldBeNil)
		})
		Convey("GetSubscriptions", func() {
			expected := fmt.Errorf("can not read subscriptions")
			database.EXPECT().GetUserSubscriptionIDs(login).Return(make([]string, 0), nil)
			database.EXPECT().GetSubscriptions(make([]string, 0)).Return(nil, expected)
			settings, err := GetUserSettings(database, login)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldBeNil)
		})
		Convey("GetUserContactIDs", func() {
			expected := fmt.Errorf("can not read contact ids")
			database.EXPECT().GetUserSubscriptionIDs(login).Return(make([]string, 0), nil)
			database.EXPECT().GetSubscriptions(make([]string, 0)).Return(make([]*moira.SubscriptionData, 0), nil)
			database.EXPECT().GetUserContactIDs(login).Return(nil, expected)
			settings, err := GetUserSettings(database, login)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldBeNil)
		})
		Convey("GetContacts", func() {
			expected := fmt.Errorf("can not read contacts")
			subscriptionIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
			subscriptions := []*moira.SubscriptionData{{ID: subscriptionIDs[0]}, {ID: subscriptionIDs[1]}}
			contactIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
			database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
			database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
			database.EXPECT().GetUserContactIDs(login).Return(contactIDs, nil)
			database.EXPECT().GetContacts(contactIDs).Return(nil, expected)
			settings, err := GetUserSettings(database, login)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldBeNil)
		})
	})
}
