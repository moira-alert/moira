package controller

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestGetUserSettings(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	login := "user"
	auth := &api.Authorization{}

	Convey("Success get user data", t, func() {
		subscriptionIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
		subscriptions := []*moira.SubscriptionData{{ID: subscriptionIDs[0]}, {ID: subscriptionIDs[1]}}

		contactIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
		contacts := []*moira.ContactData{{ID: contactIDs[0]}, {ID: contactIDs[1]}}

		emergencyContacts := []*moira.EmergencyContact{{ContactID: contactIDs[0]}, {ContactID: contactIDs[1]}}

		database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
		database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
		database.EXPECT().GetUserContactIDs(login).Return(contactIDs, nil)
		database.EXPECT().GetContacts(contactIDs).Return(contacts, nil)
		database.EXPECT().GetEmergencyContactsByIDs(contactIDs).Return(emergencyContacts, nil)

		settings, err := GetUserSettings(database, login, auth)
		So(err, ShouldBeNil)
		So(settings, ShouldResemble, &dto.UserSettings{
			User:              dto.User{Login: login},
			Contacts:          []moira.ContactData{*contacts[0], *contacts[1]},
			Subscriptions:     []moira.SubscriptionData{*subscriptions[0], *subscriptions[1]},
			EmergencyContacts: []dto.EmergencyContact{dto.EmergencyContact(*emergencyContacts[0]), dto.EmergencyContact(*emergencyContacts[1])},
		})
	})

	Convey("No contacts, subscriptions and emergency contacts", t, func() {
		database.EXPECT().GetUserSubscriptionIDs(login).Return([]string{}, nil)
		database.EXPECT().GetSubscriptions([]string{}).Return([]*moira.SubscriptionData{}, nil)
		database.EXPECT().GetUserContactIDs(login).Return([]string{}, nil)
		database.EXPECT().GetContacts([]string{}).Return([]*moira.ContactData{}, nil)
		database.EXPECT().GetEmergencyContactsByIDs([]string{}).Return([]*moira.EmergencyContact{}, nil)

		settings, err := GetUserSettings(database, login, auth)
		So(err, ShouldBeNil)
		So(settings, ShouldResemble, &dto.UserSettings{
			User:              dto.User{Login: login},
			Contacts:          make([]moira.ContactData, 0),
			Subscriptions:     make([]moira.SubscriptionData, 0),
			EmergencyContacts: make([]dto.EmergencyContact, 0),
		})
	})

	Convey("Admin auth enabled", t, func() {
		adminLogin := "admin_login"
		authFull := &api.Authorization{Enabled: true, AdminList: map[string]struct{}{adminLogin: {}}}

		Convey("User is not admin", func() {
			database.EXPECT().GetUserSubscriptionIDs(login).Return([]string{}, nil)
			database.EXPECT().GetSubscriptions([]string{}).Return([]*moira.SubscriptionData{}, nil)
			database.EXPECT().GetUserContactIDs(login).Return([]string{}, nil)
			database.EXPECT().GetContacts([]string{}).Return([]*moira.ContactData{}, nil)
			database.EXPECT().GetEmergencyContactsByIDs([]string{}).Return([]*moira.EmergencyContact{}, nil)

			settings, err := GetUserSettings(database, login, authFull)
			So(err, ShouldBeNil)
			So(settings, ShouldResemble, &dto.UserSettings{
				User:              dto.User{Login: login, Role: api.RoleUser, AuthEnabled: true},
				Contacts:          make([]moira.ContactData, 0),
				Subscriptions:     make([]moira.SubscriptionData, 0),
				EmergencyContacts: make([]dto.EmergencyContact, 0),
			})
		})

		Convey("User is admin", func() {
			database.EXPECT().GetUserSubscriptionIDs(adminLogin).Return([]string{}, nil)
			database.EXPECT().GetSubscriptions([]string{}).Return([]*moira.SubscriptionData{}, nil)
			database.EXPECT().GetUserContactIDs(adminLogin).Return([]string{}, nil)
			database.EXPECT().GetContacts([]string{}).Return([]*moira.ContactData{}, nil)
			database.EXPECT().GetEmergencyContactsByIDs([]string{}).Return([]*moira.EmergencyContact{}, nil)

			settings, err := GetUserSettings(database, adminLogin, authFull)
			So(err, ShouldBeNil)
			So(settings, ShouldResemble, &dto.UserSettings{
				User:              dto.User{Login: adminLogin, Role: api.RoleAdmin, AuthEnabled: true},
				Contacts:          make([]moira.ContactData, 0),
				Subscriptions:     make([]moira.SubscriptionData, 0),
				EmergencyContacts: make([]dto.EmergencyContact, 0),
			})
		})
	})

	Convey("Errors", t, func() {
		Convey("GetUserSubscriptionIDs", func() {
			expected := fmt.Errorf("can not read ids")
			database.EXPECT().GetUserSubscriptionIDs(login).Return(nil, expected)

			settings, err := GetUserSettings(database, login, auth)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldBeNil)
		})

		Convey("GetSubscriptions", func() {
			expected := fmt.Errorf("can not read subscriptions")
			database.EXPECT().GetUserSubscriptionIDs(login).Return([]string{}, nil)
			database.EXPECT().GetSubscriptions([]string{}).Return(nil, expected)

			settings, err := GetUserSettings(database, login, auth)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldBeNil)
		})

		Convey("GetUserContactIDs", func() {
			expected := fmt.Errorf("can not read contact ids")
			database.EXPECT().GetUserSubscriptionIDs(login).Return([]string{}, nil)
			database.EXPECT().GetSubscriptions([]string{}).Return([]*moira.SubscriptionData{}, nil)
			database.EXPECT().GetUserContactIDs(login).Return(nil, expected)

			settings, err := GetUserSettings(database, login, auth)
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

			settings, err := GetUserSettings(database, login, auth)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldBeNil)
		})

		Convey("GetEmergencyContacts", func() {
			expected := fmt.Errorf("can not read emergency contacts")
			subscriptionIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
			subscriptions := []*moira.SubscriptionData{{ID: subscriptionIDs[0]}, {ID: subscriptionIDs[1]}}
			contactIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
			contacts := []*moira.ContactData{{ID: contactIDs[0]}, {ID: contactIDs[1]}}

			database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
			database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
			database.EXPECT().GetUserContactIDs(login).Return(contactIDs, nil)
			database.EXPECT().GetContacts(contactIDs).Return(contacts, nil)
			database.EXPECT().GetEmergencyContactsByIDs(contactIDs).Return(nil, expected)
			settings, err := GetUserSettings(database, login, auth)

			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldBeNil)
		})
	})
}
