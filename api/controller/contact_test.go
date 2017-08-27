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

func TestGetAllContacts(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Error get all contacts", t, func() {
		expected := fmt.Errorf("Oooops! Can not get all contacts")
		dataBase.EXPECT().GetAllContacts().Return(nil, expected)
		contacts, err := GetAllContacts(dataBase)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(contacts, ShouldBeNil)
	})

	Convey("Get contacts", t, func() {
		contacts := []moira.ContactData{
			{
				ID:    uuid.NewV4().String(),
				Type:  "mail",
				User:  "user1",
				Value: "good@mail.com",
			},
			{
				ID:    uuid.NewV4().String(),
				Type:  "pushover",
				User:  "user2",
				Value: "ggg1",
			},
		}
		dataBase.EXPECT().GetAllContacts().Return(contacts, nil)
		actual, err := GetAllContacts(dataBase)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.ContactList{List: contacts})
	})

	Convey("No contacts", t, func() {
		dataBase.EXPECT().GetAllContacts().Return(make([]moira.ContactData, 0), nil)
		contacts, err := GetAllContacts(dataBase)
		So(err, ShouldBeNil)
		So(contacts, ShouldResemble, &dto.ContactList{List: make([]moira.ContactData, 0)})
	})
}

func TestCreateContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()
	userLogin := "user"
	contact := &dto.Contact{
		Value: "some@mail.com",
		Type:  "mail",
	}

	Convey("Success create", t, func() {
		dataBase.EXPECT().WriteContact(gomock.Any()).Return(nil)
		err := CreateContact(dataBase, contact, userLogin)
		So(err, ShouldBeNil)
	})

	Convey("Error create", t, func() {
		err := fmt.Errorf("Oooops! Can not write contact")
		dataBase.EXPECT().WriteContact(gomock.Any()).Return(err)
		expected := CreateContact(dataBase, contact, userLogin)
		So(expected, ShouldResemble, &api.ErrorResponse{
			ErrorText:      err.Error(),
			HTTPStatusCode: 500,
			StatusText:     "Internal Server Error",
			Err:            err,
		})
	})
}

func TestDeleteContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	userLogin := "user"
	contactID := uuid.NewV4().String()

	Convey("Delete contact without user subscriptions", t, func() {
		dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return(make([]string, 0), nil)
		dataBase.EXPECT().GetSubscriptions(make([]string, 0)).Return(make([]*moira.SubscriptionData, 0), nil)
		dataBase.EXPECT().DeleteContact(contactID, userLogin).Return(nil)
		dataBase.EXPECT().WriteSubscriptions(make([]*moira.SubscriptionData, 0)).Return(nil)
		err := DeleteContact(dataBase, contactID, userLogin)
		So(err, ShouldBeNil)
	})

	Convey("Delete contact without contact subscriptions", t, func() {
		subscription := &moira.SubscriptionData{
			Contacts: []string{uuid.NewV4().String()},
			ID:       uuid.NewV4().String(),
		}

		dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return([]string{subscription.ID}, nil)
		dataBase.EXPECT().GetSubscriptions([]string{subscription.ID}).Return([]*moira.SubscriptionData{subscription}, nil)
		dataBase.EXPECT().DeleteContact(contactID, userLogin).Return(nil)
		dataBase.EXPECT().WriteSubscriptions(make([]*moira.SubscriptionData, 0)).Return(nil)
		err := DeleteContact(dataBase, contactID, userLogin)
		So(err, ShouldBeNil)
	})

	Convey("Delete contact with contact subscriptions", t, func() {
		subscription := moira.SubscriptionData{
			Contacts: []string{contactID},
			ID:       uuid.NewV4().String(),
		}
		expectedSub := subscription
		expectedSub.Contacts = make([]string, 0)

		dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return([]string{subscription.ID}, nil)
		dataBase.EXPECT().GetSubscriptions([]string{subscription.ID}).Return([]*moira.SubscriptionData{&subscription}, nil)
		dataBase.EXPECT().DeleteContact(contactID, userLogin).Return(nil)
		dataBase.EXPECT().WriteSubscriptions([]*moira.SubscriptionData{&expectedSub}).Return(nil)
		err := DeleteContact(dataBase, contactID, userLogin)
		So(err, ShouldBeNil)
	})

	Convey("Error tests", t, func() {
		Convey("GetUserSubscriptionIDs", func() {
			expectedError := fmt.Errorf("Oooops! Can not read user subscription ids")
			dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return(nil, expectedError)
			err := DeleteContact(dataBase, contactID, userLogin)
			So(err, ShouldResemble, api.ErrorInternalServer(expectedError))
		})
		Convey("GetSubscriptions", func() {
			expectedError := fmt.Errorf("Oooops! Can not read user subscriptions")
			dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return(make([]string, 0), nil)
			dataBase.EXPECT().GetSubscriptions(make([]string, 0)).Return(nil, expectedError)
			err := DeleteContact(dataBase, contactID, userLogin)
			So(err, ShouldResemble, api.ErrorInternalServer(expectedError))
		})
		Convey("DeleteContact", func() {
			expectedError := fmt.Errorf("Oooops! Can not delete contact")
			dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return(make([]string, 0), nil)
			dataBase.EXPECT().GetSubscriptions(make([]string, 0)).Return(make([]*moira.SubscriptionData, 0), nil)
			dataBase.EXPECT().DeleteContact(contactID, userLogin).Return(expectedError)
			err := DeleteContact(dataBase, contactID, userLogin)
			So(err, ShouldResemble, api.ErrorInternalServer(expectedError))
		})
		Convey("WriteSubscriptions", func() {
			expectedError := fmt.Errorf("Oooops! Can not write subscriptions")
			dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return(make([]string, 0), nil)
			dataBase.EXPECT().GetSubscriptions(make([]string, 0)).Return(make([]*moira.SubscriptionData, 0), nil)
			dataBase.EXPECT().DeleteContact(contactID, userLogin).Return(nil)
			dataBase.EXPECT().WriteSubscriptions(make([]*moira.SubscriptionData, 0)).Return(expectedError)
			err := DeleteContact(dataBase, contactID, userLogin)
			So(err, ShouldResemble, api.ErrorInternalServer(expectedError))
		})
	})
}
