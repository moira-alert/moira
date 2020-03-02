package controller

import (
	"fmt"
	"strings"
	"testing"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira/internal/api"
	"github.com/moira-alert/moira/internal/api/dto"
	"github.com/moira-alert/moira/internal/database"
	mock_moira_alert "github.com/moira-alert/moira/internal/mock/moira-alert"
)

func TestGetAllContacts(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Error get all contacts", t, func() {
		expected := fmt.Errorf("oooops! Can not get all contacts")
		dataBase.EXPECT().GetAllContacts().Return(nil, expected)
		contacts, err := GetAllContacts(dataBase)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(contacts, ShouldBeNil)
	})

	Convey("Get contacts", t, func() {
		contacts := []*moira2.ContactData{
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
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.ContactList{List: contacts})
	})

	Convey("No contacts", t, func() {
		dataBase.EXPECT().GetAllContacts().Return(make([]*moira2.ContactData, 0), nil)
		contacts, err := GetAllContacts(dataBase)
		So(err, ShouldBeNil)
		So(contacts, ShouldResemble, &dto.ContactList{List: make([]*moira2.ContactData, 0)})
	})
}

func TestCreateContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()
	userLogin := "user"

	Convey("Success create", t, func() {
		contact := &dto.Contact{
			Value: "some@mail.com",
			Type:  "mail",
		}
		dataBase.EXPECT().SaveContact(gomock.Any()).Return(nil)
		err := CreateContact(dataBase, contact, userLogin)
		So(err, ShouldBeNil)
		So(contact.User, ShouldResemble, userLogin)
	})

	Convey("Success create contact with id", t, func() {
		contact := dto.Contact{
			ID:    uuid.Must(uuid.NewV4()).String(),
			Value: "some@mail.com",
			Type:  "mail",
		}
		expectedContact := moira2.ContactData{
			ID:    contact.ID,
			Value: contact.Value,
			Type:  contact.Type,
			User:  userLogin,
		}
		dataBase.EXPECT().GetContact(contact.ID).Return(moira2.ContactData{}, database.ErrNil)
		dataBase.EXPECT().SaveContact(&expectedContact).Return(nil)
		err := CreateContact(dataBase, &contact, userLogin)
		So(err, ShouldBeNil)
		So(contact.User, ShouldResemble, userLogin)
		So(contact.ID, ShouldResemble, contact.ID)
	})

	Convey("Contact exists by id", t, func() {
		contact := &dto.Contact{
			ID:    uuid.Must(uuid.NewV4()).String(),
			Value: "some@mail.com",
			Type:  "mail",
		}
		dataBase.EXPECT().GetContact(contact.ID).Return(moira2.ContactData{}, nil)
		err := CreateContact(dataBase, contact, userLogin)
		So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("contact with this ID already exists")))
	})

	Convey("Error get contact", t, func() {
		contact := &dto.Contact{
			ID:    uuid.Must(uuid.NewV4()).String(),
			Value: "some@mail.com",
			Type:  "mail",
		}
		err := fmt.Errorf("oooops! Can not write contact")
		dataBase.EXPECT().GetContact(contact.ID).Return(moira2.ContactData{}, err)
		expected := CreateContact(dataBase, contact, userLogin)
		So(expected, ShouldResemble, api.ErrorInternalServer(err))
	})

	Convey("Error save contact", t, func() {
		contact := &dto.Contact{
			Value: "some@mail.com",
			Type:  "mail",
		}
		err := fmt.Errorf("oooops! Can not write contact")
		dataBase.EXPECT().SaveContact(gomock.Any()).Return(err)
		expected := CreateContact(dataBase, contact, userLogin)
		So(expected, ShouldResemble, &api.ErrorResponse{
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

	Convey("Success update", t, func() {
		contactDTO := dto.Contact{
			Value: "some@mail.com",
			Type:  "mail",
		}
		contactID := uuid.Must(uuid.NewV4()).String()
		contact := moira2.ContactData{
			Value: contactDTO.Value,
			Type:  contactDTO.Type,
			ID:    contactID,
			User:  userLogin,
		}
		dataBase.EXPECT().SaveContact(&contact).Return(nil)
		expectedContact, err := UpdateContact(dataBase, contactDTO, moira2.ContactData{ID: contactID, User: userLogin})
		So(err, ShouldBeNil)
		So(expectedContact.User, ShouldResemble, userLogin)
		So(expectedContact.ID, ShouldResemble, contactID)
	})

	Convey("Error save", t, func() {
		contactDTO := dto.Contact{
			Value: "some@mail.com",
			Type:  "mail",
		}
		contactID := uuid.Must(uuid.NewV4()).String()
		contact := moira2.ContactData{
			Value: contactDTO.Value,
			Type:  contactDTO.Type,
			ID:    contactID,
			User:  userLogin,
		}
		err := fmt.Errorf("oooops")
		dataBase.EXPECT().SaveContact(&contact).Return(err)
		expectedContact, actual := UpdateContact(dataBase, contactDTO, contact)
		So(actual, ShouldResemble, api.ErrorInternalServer(err))
		So(expectedContact.User, ShouldResemble, contactDTO.User)
		So(expectedContact.ID, ShouldResemble, contactDTO.ID)
	})
}

func TestRemoveContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	userLogin := "user"
	contactID := uuid.Must(uuid.NewV4()).String()

	Convey("Delete contact without user subscriptions", t, func() {
		dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return(make([]string, 0), nil)
		dataBase.EXPECT().GetSubscriptions(make([]string, 0)).Return(make([]*moira2.SubscriptionData, 0), nil)
		dataBase.EXPECT().RemoveContact(contactID).Return(nil)
		err := RemoveContact(dataBase, contactID, userLogin)
		So(err, ShouldBeNil)
	})

	Convey("Delete contact without contact subscriptions", t, func() {
		subscription := &moira2.SubscriptionData{
			Contacts: []string{uuid.Must(uuid.NewV4()).String()},
			ID:       uuid.Must(uuid.NewV4()).String(),
		}

		dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return([]string{subscription.ID}, nil)
		dataBase.EXPECT().GetSubscriptions([]string{subscription.ID}).Return([]*moira2.SubscriptionData{subscription}, nil)
		dataBase.EXPECT().RemoveContact(contactID).Return(nil)
		err := RemoveContact(dataBase, contactID, userLogin)
		So(err, ShouldBeNil)
	})

	Convey("Error tests", t, func() {
		Convey("GetUserSubscriptionIDs", func() {
			expectedError := fmt.Errorf("oooops! Can not read user subscription ids")
			dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return(nil, expectedError)
			err := RemoveContact(dataBase, contactID, userLogin)
			So(err, ShouldResemble, api.ErrorInternalServer(expectedError))
		})
		Convey("GetSubscriptions", func() {
			expectedError := fmt.Errorf("oooops! Can not read user subscriptions")
			dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return(make([]string, 0), nil)
			dataBase.EXPECT().GetSubscriptions(make([]string, 0)).Return(nil, expectedError)
			err := RemoveContact(dataBase, contactID, userLogin)
			So(err, ShouldResemble, api.ErrorInternalServer(expectedError))
		})
		Convey("Subscription has contact", func() {
			subscription := moira2.SubscriptionData{
				Contacts: []string{contactID},
				ID:       uuid.Must(uuid.NewV4()).String(),
				Tags:     []string{"Tag1", "Tag2"},
			}
			subscriptionSubstring := fmt.Sprintf("%s (tags: %s)", subscription.ID, strings.Join(subscription.Tags, ", "))
			expectedError := fmt.Errorf("this contact is being used in following subscriptions: %s", subscriptionSubstring)
			dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return([]string{subscription.ID}, nil)
			dataBase.EXPECT().GetSubscriptions([]string{subscription.ID}).Return([]*moira2.SubscriptionData{&subscription}, nil)
			err := RemoveContact(dataBase, contactID, userLogin)
			So(err, ShouldResemble, api.ErrorInvalidRequest(expectedError))
		})
	})
}

func TestSendTestContactNotification(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	id := uuid.Must(uuid.NewV4()).String()

	Convey("Success", t, func() {
		dataBase.EXPECT().PushNotificationEvent(gomock.Any(), false).Return(nil)
		err := SendTestContactNotification(dataBase, id)
		So(err, ShouldBeNil)
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("oooops! Can not push event")
		dataBase.EXPECT().PushNotificationEvent(gomock.Any(), false).Return(expected)
		err := SendTestContactNotification(dataBase, id)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestCheckUserPermissionsForContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	userLogin := uuid.Must(uuid.NewV4()).String()
	id := uuid.Must(uuid.NewV4()).String()

	Convey("No contact", t, func() {
		dataBase.EXPECT().GetContact(id).Return(moira2.ContactData{}, database.ErrNil)
		expectedContact, expected := CheckUserPermissionsForContact(dataBase, id, userLogin)
		So(expected, ShouldResemble, api.ErrorNotFound(fmt.Sprintf("contact with ID '%s' does not exists", id)))
		So(expectedContact, ShouldResemble, moira2.ContactData{})
	})

	Convey("Different user", t, func() {
		dataBase.EXPECT().GetContact(id).Return(moira2.ContactData{User: "diffUser"}, nil)
		expectedContact, expected := CheckUserPermissionsForContact(dataBase, id, userLogin)
		So(expected, ShouldResemble, api.ErrorForbidden("you are not permitted"))
		So(expectedContact, ShouldResemble, moira2.ContactData{User: "diffUser"})
	})

	Convey("Has contact", t, func() {
		actualContact := moira2.ContactData{ID: id, User: userLogin}
		dataBase.EXPECT().GetContact(id).Return(actualContact, nil)
		expectedContact, expected := CheckUserPermissionsForContact(dataBase, id, userLogin)
		So(expected, ShouldBeNil)
		So(expectedContact, ShouldResemble, actualContact)
	})

	Convey("Error get contact", t, func() {
		err := fmt.Errorf("oooops! Can not read contact")
		dataBase.EXPECT().GetContact(id).Return(moira2.ContactData{User: userLogin}, err)
		expectedContact, expected := CheckUserPermissionsForContact(dataBase, id, userLogin)
		So(expected, ShouldResemble, api.ErrorInternalServer(err))
		So(expectedContact, ShouldResemble, moira2.ContactData{User: userLogin})
	})
}
