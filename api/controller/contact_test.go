package controller

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
)

func TestGetAllContacts(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Error get all contacts", t, func(c C) {
		expected := fmt.Errorf("oooops! Can not get all contacts")
		dataBase.EXPECT().GetAllContacts().Return(nil, expected)
		contacts, err := GetAllContacts(dataBase)
		c.So(err, ShouldResemble, api.ErrorInternalServer(expected))
		c.So(contacts, ShouldBeNil)
	})

	Convey("Get contacts", t, func(c C) {
		contacts := []*moira.ContactData{
			{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Type:  "mail",
				User:  "user1",
				Value: "good@mail.com",
			},
			{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Type:  "pushover",
				User:  "user2",
				Value: "ggg1",
			},
		}
		dataBase.EXPECT().GetAllContacts().Return(contacts, nil)
		actual, err := GetAllContacts(dataBase)
		c.So(err, ShouldBeNil)
		c.So(actual, ShouldResemble, &dto.ContactList{List: contacts})
	})

	Convey("No contacts", t, func(c C) {
		dataBase.EXPECT().GetAllContacts().Return(make([]*moira.ContactData, 0), nil)
		contacts, err := GetAllContacts(dataBase)
		c.So(err, ShouldBeNil)
		c.So(contacts, ShouldResemble, &dto.ContactList{List: make([]*moira.ContactData, 0)})
	})
}

func TestCreateContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()
	userLogin := "user"

	Convey("Success create", t, func(c C) {
		contact := &dto.Contact{
			Value: "some@mail.com",
			Type:  "mail",
		}
		dataBase.EXPECT().SaveContact(gomock.Any()).Return(nil)
		err := CreateContact(dataBase, contact, userLogin)
		c.So(err, ShouldBeNil)
		c.So(contact.User, ShouldResemble, userLogin)
	})

	Convey("Success create contact with id", t, func(c C) {
		contact := dto.Contact{
			ID:    uuid.Must(uuid.NewV4()).String(),
			Value: "some@mail.com",
			Type:  "mail",
		}
		expectedContact := moira.ContactData{
			ID:    contact.ID,
			Value: contact.Value,
			Type:  contact.Type,
			User:  userLogin,
		}
		dataBase.EXPECT().GetContact(contact.ID).Return(moira.ContactData{}, database.ErrNil)
		dataBase.EXPECT().SaveContact(&expectedContact).Return(nil)
		err := CreateContact(dataBase, &contact, userLogin)
		c.So(err, ShouldBeNil)
		c.So(contact.User, ShouldResemble, userLogin)
		c.So(contact.ID, ShouldResemble, contact.ID)
	})

	Convey("Contact exists by id", t, func(c C) {
		contact := &dto.Contact{
			ID:    uuid.Must(uuid.NewV4()).String(),
			Value: "some@mail.com",
			Type:  "mail",
		}
		dataBase.EXPECT().GetContact(contact.ID).Return(moira.ContactData{}, nil)
		err := CreateContact(dataBase, contact, userLogin)
		c.So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("contact with this ID already exists")))
	})

	Convey("Error get contact", t, func(c C) {
		contact := &dto.Contact{
			ID:    uuid.Must(uuid.NewV4()).String(),
			Value: "some@mail.com",
			Type:  "mail",
		}
		err := fmt.Errorf("oooops! Can not write contact")
		dataBase.EXPECT().GetContact(contact.ID).Return(moira.ContactData{}, err)
		expected := CreateContact(dataBase, contact, userLogin)
		c.So(expected, ShouldResemble, api.ErrorInternalServer(err))
	})

	Convey("Error save contact", t, func(c C) {
		contact := &dto.Contact{
			Value: "some@mail.com",
			Type:  "mail",
		}
		err := fmt.Errorf("oooops! Can not write contact")
		dataBase.EXPECT().SaveContact(gomock.Any()).Return(err)
		expected := CreateContact(dataBase, contact, userLogin)
		c.So(expected, ShouldResemble, &api.ErrorResponse{
			ErrorText:      err.Error(),
			HTTPStatusCode: 500,
			StatusText:     "Internal Server Error",
			Err:            err,
		})
	})
}

func TestUpdateContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()
	userLogin := "user"

	Convey("Success update", t, func(c C) {
		contactDTO := dto.Contact{
			Value: "some@mail.com",
			Type:  "mail",
		}
		contactID := uuid.Must(uuid.NewV4()).String()
		contact := moira.ContactData{
			Value: contactDTO.Value,
			Type:  contactDTO.Type,
			ID:    contactID,
			User:  userLogin,
		}
		dataBase.EXPECT().SaveContact(&contact).Return(nil)
		expectedContact, err := UpdateContact(dataBase, contactDTO, moira.ContactData{ID: contactID, User: userLogin})
		c.So(err, ShouldBeNil)
		c.So(expectedContact.User, ShouldResemble, userLogin)
		c.So(expectedContact.ID, ShouldResemble, contactID)
	})

	Convey("Error save", t, func(c C) {
		contactDTO := dto.Contact{
			Value: "some@mail.com",
			Type:  "mail",
		}
		contactID := uuid.Must(uuid.NewV4()).String()
		contact := moira.ContactData{
			Value: contactDTO.Value,
			Type:  contactDTO.Type,
			ID:    contactID,
			User:  userLogin,
		}
		err := fmt.Errorf("oooops")
		dataBase.EXPECT().SaveContact(&contact).Return(err)
		expectedContact, actual := UpdateContact(dataBase, contactDTO, contact)
		c.So(actual, ShouldResemble, api.ErrorInternalServer(err))
		c.So(expectedContact.User, ShouldResemble, contactDTO.User)
		c.So(expectedContact.ID, ShouldResemble, contactDTO.ID)
	})
}

func TestRemoveContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	userLogin := "user"
	contactID := uuid.Must(uuid.NewV4()).String()

	Convey("Delete contact without user subscriptions", t, func(c C) {
		dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return(make([]string, 0), nil)
		dataBase.EXPECT().GetSubscriptions(make([]string, 0)).Return(make([]*moira.SubscriptionData, 0), nil)
		dataBase.EXPECT().RemoveContact(contactID).Return(nil)
		err := RemoveContact(dataBase, contactID, userLogin)
		c.So(err, ShouldBeNil)
	})

	Convey("Delete contact without contact subscriptions", t, func(c C) {
		subscription := &moira.SubscriptionData{
			Contacts: []string{uuid.Must(uuid.NewV4()).String()},
			ID:       uuid.Must(uuid.NewV4()).String(),
		}

		dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return([]string{subscription.ID}, nil)
		dataBase.EXPECT().GetSubscriptions([]string{subscription.ID}).Return([]*moira.SubscriptionData{subscription}, nil)
		dataBase.EXPECT().RemoveContact(contactID).Return(nil)
		err := RemoveContact(dataBase, contactID, userLogin)
		c.So(err, ShouldBeNil)
	})

	Convey("Error tests", t, func(c C) {
		Convey("GetUserSubscriptionIDs", t, func(c C) {
			expectedError := fmt.Errorf("oooops! Can not read user subscription ids")
			dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return(nil, expectedError)
			err := RemoveContact(dataBase, contactID, userLogin)
			c.So(err, ShouldResemble, api.ErrorInternalServer(expectedError))
		})
		Convey("GetSubscriptions", t, func(c C) {
			expectedError := fmt.Errorf("oooops! Can not read user subscriptions")
			dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return(make([]string, 0), nil)
			dataBase.EXPECT().GetSubscriptions(make([]string, 0)).Return(nil, expectedError)
			err := RemoveContact(dataBase, contactID, userLogin)
			c.So(err, ShouldResemble, api.ErrorInternalServer(expectedError))
		})
		Convey("Subscription has contact", t, func(c C) {
			subscription := moira.SubscriptionData{
				Contacts: []string{contactID},
				ID:       uuid.Must(uuid.NewV4()).String(),
				Tags:     []string{"Tag1", "Tag2"},
			}
			subscriptionSubstring := fmt.Sprintf("%s (tags: %s)", subscription.ID, strings.Join(subscription.Tags, ", "))
			expectedError := fmt.Errorf("this contact is being used in following subscriptions: %s", subscriptionSubstring)
			dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return([]string{subscription.ID}, nil)
			dataBase.EXPECT().GetSubscriptions([]string{subscription.ID}).Return([]*moira.SubscriptionData{&subscription}, nil)
			err := RemoveContact(dataBase, contactID, userLogin)
			c.So(err, ShouldResemble, api.ErrorInvalidRequest(expectedError))
		})
	})
}

func TestSendTestContactNotification(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	id := uuid.Must(uuid.NewV4()).String()

	Convey("Success", t, func(c C) {
		dataBase.EXPECT().PushNotificationEvent(gomock.Any(), false).Return(nil)
		err := SendTestContactNotification(dataBase, id)
		c.So(err, ShouldBeNil)
	})

	Convey("Error", t, func(c C) {
		expected := fmt.Errorf("oooops! Can not push event")
		dataBase.EXPECT().PushNotificationEvent(gomock.Any(), false).Return(expected)
		err := SendTestContactNotification(dataBase, id)
		c.So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestCheckUserPermissionsForContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	userLogin := uuid.Must(uuid.NewV4()).String()
	id := uuid.Must(uuid.NewV4()).String()

	Convey("No contact", t, func(c C) {
		dataBase.EXPECT().GetContact(id).Return(moira.ContactData{}, database.ErrNil)
		expectedContact, expected := CheckUserPermissionsForContact(dataBase, id, userLogin)
		c.So(expected, ShouldResemble, api.ErrorNotFound(fmt.Sprintf("contact with ID '%s' does not exists", id)))
		c.So(expectedContact, ShouldResemble, moira.ContactData{})
	})

	Convey("Different user", t, func(c C) {
		dataBase.EXPECT().GetContact(id).Return(moira.ContactData{User: "diffUser"}, nil)
		expectedContact, expected := CheckUserPermissionsForContact(dataBase, id, userLogin)
		c.So(expected, ShouldResemble, api.ErrorForbidden("you are not permitted"))
		c.So(expectedContact, ShouldResemble, moira.ContactData{User: "diffUser"})
	})

	Convey("Has contact", t, func(c C) {
		actualContact := moira.ContactData{ID: id, User: userLogin}
		dataBase.EXPECT().GetContact(id).Return(actualContact, nil)
		expectedContact, expected := CheckUserPermissionsForContact(dataBase, id, userLogin)
		c.So(expected, ShouldBeNil)
		c.So(expectedContact, ShouldResemble, actualContact)
	})

	Convey("Error get contact", t, func(c C) {
		err := fmt.Errorf("oooops! Can not read contact")
		dataBase.EXPECT().GetContact(id).Return(moira.ContactData{User: userLogin}, err)
		expectedContact, expected := CheckUserPermissionsForContact(dataBase, id, userLogin)
		c.So(expected, ShouldResemble, api.ErrorInternalServer(err))
		c.So(expectedContact, ShouldResemble, moira.ContactData{User: userLogin})
	})
}
