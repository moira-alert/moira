package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	db "github.com/moira-alert/moira/database"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	ContactIDKey   = "contactID"
	ContactKey     = "contact"
	LoginKey       = "login"
	defaultContact = "testContact"
	defaultLogin   = "testLogin"
)

func TestGetAllContacts(t *testing.T) {
	Convey("Test get all contacts", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		testErr := errors.New("test error")

		Convey("Correctly returns all contacts", func() {
			mockDb.EXPECT().GetAllContacts().Return([]*moira.ContactData{
				{
					ID:    defaultContact,
					Type:  "mail",
					Value: "moira@skbkontur.ru",
					User:  "moira",
					Team:  "",
				},
			}, nil).Times(1)
			database = mockDb

			expected := &dto.ContactList{
				List: []*moira.ContactData{
					{
						ID:    defaultContact,
						Type:  "mail",
						Value: "moira@skbkontur.ru",
						User:  "moira",
						Team:  "",
					},
				},
			}

			testRequest := httptest.NewRequest(http.MethodGet, "/contact", nil)

			getAllContacts(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			actual := &dto.ContactList{}
			err := json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Internal server error when trying to get all contacts", func() {
			mockDb.EXPECT().GetAllContacts().Return(nil, testErr).Times(1)
			database = mockDb

			expected := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  testErr.Error(),
			}

			testRequest := httptest.NewRequest(http.MethodGet, "/contact", nil)

			getAllContacts(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err := json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestGetContactById(t *testing.T) {
	Convey("Test get contact by id", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		testErr := errors.New("test error")

		Convey("Correctly returns contact by id", func() {
			contactID := defaultContact
			mockDb.EXPECT().GetContact(contactID).Return(moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
				User:  "test",
				Team:  "",
			}, nil).Times(1)
			database = mockDb

			expected := &dto.Contact{
				ID:     contactID,
				Type:   "mail",
				Value:  "moira@skbkontur.ru",
				User:   "test",
				TeamID: "",
			}

			testRequest := httptest.NewRequest(http.MethodGet, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactIDKey, contactID))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactKey, moira.ContactData{ID: contactID}))

			getContactById(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &dto.Contact{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Internal server error when trying to get a contact by id", func() {
			contactID := defaultContact
			mockDb.EXPECT().GetContact(contactID).Return(moira.ContactData{}, testErr).Times(1)
			database = mockDb

			expected := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  testErr.Error(),
			}

			testRequest := httptest.NewRequest(http.MethodGet, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactIDKey, contactID))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactKey, moira.ContactData{ID: contactID}))

			getContactById(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestCreateNewContact(t *testing.T) {
	Convey("Test create new contact", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		login := defaultLogin
		testErr := errors.New("test error")

		newContactDto := &dto.Contact{
			ID:     defaultContact,
			Type:   "mail",
			Value:  "moira@skbkontur.ru",
			User:   login,
			TeamID: "",
		}

		Convey("Correctly create new contact with the given id", func() {
			jsonContact, err := json.Marshal(newContactDto)
			So(err, ShouldBeNil)

			mockDb.EXPECT().GetContact(defaultContact).Return(moira.ContactData{}, db.ErrNil).Times(1)
			mockDb.EXPECT().SaveContact(&moira.ContactData{
				ID:    newContactDto.ID,
				Type:  newContactDto.Type,
				Value: newContactDto.Value,
				User:  newContactDto.User,
				Team:  newContactDto.TeamID,
			}).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPut, "/contact", bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), LoginKey, login))
			testRequest.Header.Add("content-type", "application/json")

			createNewContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &dto.Contact{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, newContactDto)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Correctly create new contact without given id", func() {
			newContactDto.ID = ""
			defer func() {
				newContactDto.ID = defaultContact
			}()

			jsonContact, err := json.Marshal(newContactDto)
			So(err, ShouldBeNil)

			mockDb.EXPECT().SaveContact(gomock.Any()).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPut, "/contact", bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), LoginKey, login))
			testRequest.Header.Add("content-type", "application/json")

			createNewContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &dto.Contact{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual.TeamID, ShouldEqual, newContactDto.TeamID)
			So(actual.Type, ShouldEqual, newContactDto.Type)
			So(actual.User, ShouldEqual, newContactDto.User)
			So(actual.Value, ShouldEqual, newContactDto.Value)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Trying to create a new contact with the id of an existing contact", func() {
			expected := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  "contact with this ID already exists",
			}
			jsonContact, err := json.Marshal(newContactDto)
			So(err, ShouldBeNil)

			mockDb.EXPECT().GetContact(newContactDto.ID).Return(moira.ContactData{
				ID:    newContactDto.ID,
				Type:  newContactDto.Type,
				Value: newContactDto.Value,
				User:  newContactDto.User,
				Team:  newContactDto.TeamID,
			}, nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPut, "/contact", bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), LoginKey, login))
			testRequest.Header.Add("content-type", "application/json")

			createNewContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Internal error when trying to create a new contact with id", func() {
			expected := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  testErr.Error(),
			}
			jsonContact, err := json.Marshal(newContactDto)
			So(err, ShouldBeNil)

			mockDb.EXPECT().GetContact(newContactDto.ID).Return(moira.ContactData{}, testErr).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPut, "/contact", bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), LoginKey, login))
			testRequest.Header.Add("content-type", "application/json")

			createNewContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})

		Convey("Trying to create a contact when both userLogin and teamID specified", func() {
			newContactDto.TeamID = "test"
			defer func() {
				newContactDto.TeamID = ""
			}()

			expected := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  "CreateContact: cannot create contact when both userLogin and teamID specified",
			}
			jsonContact, err := json.Marshal(newContactDto)
			So(err, ShouldBeNil)

			testRequest := httptest.NewRequest(http.MethodPut, "/contact", bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), LoginKey, login))
			testRequest.Header.Add("content-type", "application/json")

			createNewContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestUpdateContact(t *testing.T) {
	Convey("Test update contact", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		testErr := errors.New("test error")
		contactID := defaultContact
		updatedContactDto := &dto.Contact{
			ID:     contactID,
			Type:   "mail",
			Value:  "moira@skbkontur.ru",
			User:   "test",
			TeamID: "",
		}

		Convey("Successful contact updated", func() {
			jsonContact, err := json.Marshal(updatedContactDto)
			So(err, ShouldBeNil)

			mockDb.EXPECT().SaveContact(&moira.ContactData{
				ID:    updatedContactDto.ID,
				Type:  updatedContactDto.Type,
				Value: updatedContactDto.Value,
				User:  updatedContactDto.User,
			}).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPut, "/contact/"+contactID, bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactKey, moira.ContactData{
				ID:    contactID,
				Type:  updatedContactDto.Type,
				Value: updatedContactDto.Value,
				User:  updatedContactDto.User,
			}))
			testRequest.Header.Add("content-type", "application/json")

			updateContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &dto.Contact{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, updatedContactDto)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Internal error when trying to update contact", func() {
			expected := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  testErr.Error(),
			}
			jsonContact, err := json.Marshal(updatedContactDto)
			So(err, ShouldBeNil)

			mockDb.EXPECT().SaveContact(&moira.ContactData{
				ID:    updatedContactDto.ID,
				Type:  updatedContactDto.Type,
				Value: updatedContactDto.Value,
				User:  updatedContactDto.User,
			}).Return(testErr).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPut, "/contact/"+contactID, bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactKey, moira.ContactData{
				ID:    contactID,
				Type:  updatedContactDto.Type,
				Value: updatedContactDto.Value,
				User:  updatedContactDto.User,
			}))
			testRequest.Header.Add("content-type", "application/json")

			updateContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestRemoveContact(t *testing.T) {
	Convey("Test remove contact", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		testErr := errors.New("test error")
		contactID := defaultContact

		Convey("Successful deletion of a contact without user, team id and subscriptions", func() {
			mockDb.EXPECT().GetSubscriptions([]string{}).Return([]*moira.SubscriptionData{}, nil).Times(1)
			mockDb.EXPECT().RemoveContact(contactID).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
			}))
			testRequest.Header.Add("content-type", "application/json")

			removeContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			actual := contentBytes

			So(actual, ShouldResemble, []byte{})
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Successful deletion of a contact without team id and subscriptions", func() {
			mockDb.EXPECT().GetUserSubscriptionIDs("test").Return([]string{}, nil).Times(1)
			mockDb.EXPECT().GetSubscriptions([]string{}).Return([]*moira.SubscriptionData{}, nil).Times(1)
			mockDb.EXPECT().RemoveContact(contactID).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
				User:  "test",
			}))
			testRequest.Header.Add("content-type", "application/json")

			removeContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			actual := contentBytes

			So(actual, ShouldResemble, []byte{})
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Successful deletion of a contact without user id and subscriptions", func() {
			mockDb.EXPECT().GetTeamSubscriptionIDs("test").Return([]string{}, nil).Times(1)
			mockDb.EXPECT().GetSubscriptions([]string{}).Return([]*moira.SubscriptionData{}, nil).Times(1)
			mockDb.EXPECT().RemoveContact(contactID).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
				Team:  "test",
			}))
			testRequest.Header.Add("content-type", "application/json")

			removeContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			actual := contentBytes

			So(actual, ShouldResemble, []byte{})
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Successful deletion of a contact without subscriptions", func() {
			mockDb.EXPECT().GetUserSubscriptionIDs("test").Return([]string{}, nil).Times(1)
			mockDb.EXPECT().GetTeamSubscriptionIDs("test").Return([]string{}, nil).Times(1)
			mockDb.EXPECT().GetSubscriptions([]string{}).Return([]*moira.SubscriptionData{}, nil).Times(1)
			mockDb.EXPECT().RemoveContact(contactID).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
				User:  "test",
				Team:  "test",
			}))
			testRequest.Header.Add("content-type", "application/json")

			removeContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			actual := contentBytes

			So(actual, ShouldResemble, []byte{})
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Error when deleting a contact, the user has existing subscriptions", func() {
			expected := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  "this contact is being used in following subscriptions:  (tags: test)",
			}

			mockDb.EXPECT().GetUserSubscriptionIDs("test").Return([]string{"test"}, nil).Times(1)
			mockDb.EXPECT().GetSubscriptions([]string{"test"}).Return([]*moira.SubscriptionData{
				{
					Contacts: []string{"testContact"},
					Tags:     []string{"test"},
				},
			}, nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
				User:  "test",
			}))
			testRequest.Header.Add("content-type", "application/json")

			removeContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Error when deleting a contact, the team has existing subscriptions", func() {
			expected := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  "this contact is being used in following subscriptions:  (tags: test)",
			}

			mockDb.EXPECT().GetTeamSubscriptionIDs("test").Return([]string{"test"}, nil).Times(1)
			mockDb.EXPECT().GetSubscriptions([]string{"test"}).Return([]*moira.SubscriptionData{
				{
					Contacts: []string{"testContact"},
					Tags:     []string{"test"},
				},
			}, nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
				Team:  "test",
			}))
			testRequest.Header.Add("content-type", "application/json")

			removeContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Error when deleting a contact, the user and team has existing subscriptions", func() {
			expected := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  "this contact is being used in following subscriptions:  (tags: test1),  (tags: test2)",
			}

			mockDb.EXPECT().GetUserSubscriptionIDs("test1").Return([]string{"test1"}, nil).Times(1)
			mockDb.EXPECT().GetTeamSubscriptionIDs("test2").Return([]string{"test2"}, nil).Times(1)
			mockDb.EXPECT().GetSubscriptions([]string{"test1", "test2"}).Return([]*moira.SubscriptionData{
				{
					Contacts: []string{"testContact"},
					Tags:     []string{"test1"},
				},
				{
					Contacts: []string{"testContact"},
					Tags:     []string{"test2"},
				},
			}, nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
				Team:  "test2",
				User:  "test1",
			}))
			testRequest.Header.Add("content-type", "application/json")

			removeContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Internal server error when deleting of a contact without user, team id and subscriptions", func() {
			expected := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  testErr.Error(),
			}

			mockDb.EXPECT().GetSubscriptions([]string{}).Return([]*moira.SubscriptionData{}, nil).Times(1)
			mockDb.EXPECT().RemoveContact(contactID).Return(testErr).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
			}))
			testRequest.Header.Add("content-type", "application/json")

			removeContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestSendTestContactNotification(t *testing.T) {
	Convey("Test send test contact notification", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		testErr := errors.New("test error")
		contactID := defaultContact

		Convey("Successful send test contact notification", func() {
			mockDb.EXPECT().PushNotificationEvent(gomock.Any(), false).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPost, "/contact/"+contactID+"/test", nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactIDKey, contactID))

			sendTestContactNotification(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			actual := contentBytes

			So(actual, ShouldResemble, []byte{})
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Internal server error when sendin test contact notification", func() {
			expected := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  testErr.Error(),
			}

			mockDb.EXPECT().PushNotificationEvent(gomock.Any(), false).Return(testErr).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPost, "/contact/"+contactID+"/test", nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactIDKey, contactID))

			sendTestContactNotification(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})
	})
}
