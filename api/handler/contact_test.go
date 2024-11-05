package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	db "github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/datatypes"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

const (
	testContactIDKey        = "contactID"
	testContactKey          = "contact"
	testAuthKey             = "auth"
	testLoginKey            = "login"
	testContactsTemplateKey = "contactsTemplate"
	defaultContact          = "testContact"
	defaultLogin            = "testLogin"
	defaultTeamID           = "testTeamID"
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
					User:  defaultLogin,
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
						User:  defaultLogin,
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
				User:  defaultLogin,
				Team:  "",
			}, nil).Times(1)
			database = mockDb

			expected := &dto.Contact{
				ID:     contactID,
				Type:   "mail",
				Value:  "moira@skbkontur.ru",
				User:   defaultLogin,
				TeamID: "",
			}

			testRequest := httptest.NewRequest(http.MethodGet, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactIDKey, contactID))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{ID: contactID}))

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
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactIDKey, contactID))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{ID: contactID}))

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

		auth := &api.Authorization{
			Enabled: false,
			AllowedContactTypes: map[string]struct{}{
				"mail": {},
			},
		}

		contactsTemplate := []api.WebContact{
			{
				ContactType:     "mail",
				ValidationRegex: "@skbkontur.ru",
			},
		}

		newContactDto := &dto.Contact{
			ID:     defaultContact,
			Name:   "Mail Alerts",
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
				Name:  newContactDto.Name,
				Type:  newContactDto.Type,
				Value: newContactDto.Value,
				User:  newContactDto.User,
				Team:  newContactDto.TeamID,
			}).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPut, "/contact", bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testLoginKey, login))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
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
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testLoginKey, login))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
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
			So(actual.Name, ShouldEqual, newContactDto.Name)
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
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testLoginKey, login))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
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
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testLoginKey, login))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
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

		Convey("Invalid request when trying to create a new contact with invalid value", func() {
			expected := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  "contact value doesn't match regex: '@yandex.ru'",
			}
			jsonContact, err := json.Marshal(newContactDto)
			So(err, ShouldBeNil)

			contactsTemplate = []api.WebContact{
				{
					ContactType:     "mail",
					ValidationRegex: "@yandex.ru",
				},
			}

			mockDb.EXPECT().GetContact(newContactDto.ID).Return(moira.ContactData{}, db.ErrNil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPut, "/contact", bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testLoginKey, login))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
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

		contactsTemplate = []api.WebContact{
			{
				ContactType:     "mail",
				ValidationRegex: "@skbkontur.ru",
			},
		}

		Convey("Trying to create a contact when both userLogin and teamID specified", func() {
			newContactDto.TeamID = defaultTeamID
			defer func() {
				newContactDto.TeamID = ""
			}()

			expected := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  "contact cannot have both the user field and the team_id field filled in",
			}
			jsonContact, err := json.Marshal(newContactDto)
			So(err, ShouldBeNil)

			testRequest := httptest.NewRequest(http.MethodPut, "/contact", bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testLoginKey, login))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
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
			Name:   "Mail Alerts",
			Type:   "mail",
			Value:  "moira@skbkontur.ru",
			User:   defaultLogin,
			TeamID: "",
		}

		contactsTemplate := []api.WebContact{
			{
				ContactType:     "mail",
				ValidationRegex: "@skbkontur.ru",
			},
		}

		Convey("Successful contact updated", func() {
			jsonContact, err := json.Marshal(updatedContactDto)
			So(err, ShouldBeNil)

			mockDb.EXPECT().SaveContact(&moira.ContactData{
				ID:    updatedContactDto.ID,
				Name:  updatedContactDto.Name,
				Type:  updatedContactDto.Type,
				Value: updatedContactDto.Value,
				User:  updatedContactDto.User,
			}).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPut, "/contact/"+contactID, bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{
				ID:    contactID,
				Name:  updatedContactDto.Name,
				Type:  updatedContactDto.Type,
				Value: updatedContactDto.Value,
				User:  updatedContactDto.User,
			}))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, &api.Authorization{
				AllowedContactTypes: map[string]struct{}{
					updatedContactDto.Type: {},
				},
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

		Convey("Failed to update a contact with the specified user and team field", func() {
			updatedContactDto.TeamID = defaultTeamID
			defer func() {
				updatedContactDto.TeamID = ""
			}()

			jsonContact, err := json.Marshal(updatedContactDto)
			So(err, ShouldBeNil)

			testRequest := httptest.NewRequest(http.MethodPut, "/contact/"+contactID, bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{
				ID:    contactID,
				Name:  updatedContactDto.Name,
				Type:  updatedContactDto.Type,
				Value: updatedContactDto.Value,
				User:  updatedContactDto.User,
			}))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, &api.Authorization{
				AllowedContactTypes: map[string]struct{}{
					updatedContactDto.Type: {},
				},
			}))

			testRequest.Header.Add("content-type", "application/json")

			updateContact(responseWriter, testRequest)

			expected := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  "contact cannot have both the user field and the team_id field filled in",
			}
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

		Convey("Invalid request when trying to update contact with invalid value", func() {
			expected := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  "contact value doesn't match regex: '@yandex.ru'",
			}
			jsonContact, err := json.Marshal(updatedContactDto)
			So(err, ShouldBeNil)

			contactsTemplate = []api.WebContact{
				{
					ContactType:     "mail",
					ValidationRegex: "@yandex.ru",
				},
			}

			testRequest := httptest.NewRequest(http.MethodPut, "/contact", bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{
				ID:    contactID,
				Name:  updatedContactDto.Name,
				Type:  updatedContactDto.Type,
				Value: updatedContactDto.Value,
				User:  updatedContactDto.User,
			}))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, &api.Authorization{
				AllowedContactTypes: map[string]struct{}{
					updatedContactDto.Type: {},
				},
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
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		contactsTemplate = []api.WebContact{
			{
				ContactType:     "mail",
				ValidationRegex: "@skbkontur.ru",
			},
		}

		Convey("Internal error when trying to update contact", func() {
			expected := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  testErr.Error(),
			}
			jsonContact, err := json.Marshal(updatedContactDto)
			So(err, ShouldBeNil)

			mockDb.EXPECT().SaveContact(&moira.ContactData{
				ID:    updatedContactDto.ID,
				Name:  updatedContactDto.Name,
				Type:  updatedContactDto.Type,
				Value: updatedContactDto.Value,
				User:  updatedContactDto.User,
			}).Return(testErr).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPut, "/contact/"+contactID, bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{
				ID:    contactID,
				Type:  updatedContactDto.Type,
				Value: updatedContactDto.Value,
				User:  updatedContactDto.User,
			}))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, &api.Authorization{
				AllowedContactTypes: map[string]struct{}{
					updatedContactDto.Type: {},
				},
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
			mockDb.EXPECT().GetEmergencyContact(contactID).Return(datatypes.EmergencyContact{}, db.ErrNil).Times(1)
			mockDb.EXPECT().GetSubscriptions([]string{}).Return([]*moira.SubscriptionData{}, nil).Times(1)
			mockDb.EXPECT().RemoveContact(contactID).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{
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
			mockDb.EXPECT().GetUserSubscriptionIDs(defaultLogin).Return([]string{}, nil).Times(1)
			mockDb.EXPECT().GetEmergencyContact(contactID).Return(datatypes.EmergencyContact{}, db.ErrNil).Times(1)
			mockDb.EXPECT().GetSubscriptions([]string{}).Return([]*moira.SubscriptionData{}, nil).Times(1)
			mockDb.EXPECT().RemoveContact(contactID).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
				User:  defaultLogin,
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
			mockDb.EXPECT().GetTeamSubscriptionIDs(defaultTeamID).Return([]string{}, nil).Times(1)
			mockDb.EXPECT().GetEmergencyContact(contactID).Return(datatypes.EmergencyContact{}, db.ErrNil).Times(1)
			mockDb.EXPECT().GetSubscriptions([]string{}).Return([]*moira.SubscriptionData{}, nil).Times(1)
			mockDb.EXPECT().RemoveContact(contactID).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
				Team:  defaultTeamID,
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
			mockDb.EXPECT().GetUserSubscriptionIDs(defaultLogin).Return([]string{}, nil).Times(1)
			mockDb.EXPECT().GetTeamSubscriptionIDs(defaultTeamID).Return([]string{}, nil).Times(1)
			mockDb.EXPECT().GetEmergencyContact(contactID).Return(datatypes.EmergencyContact{}, db.ErrNil).Times(1)
			mockDb.EXPECT().GetSubscriptions([]string{}).Return([]*moira.SubscriptionData{}, nil).Times(1)
			mockDb.EXPECT().RemoveContact(contactID).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
				User:  defaultLogin,
				Team:  defaultTeamID,
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

			mockDb.EXPECT().GetUserSubscriptionIDs(defaultLogin).Return([]string{"test"}, nil).Times(1)
			mockDb.EXPECT().GetEmergencyContact(contactID).Return(datatypes.EmergencyContact{}, db.ErrNil).Times(1)
			mockDb.EXPECT().GetSubscriptions([]string{"test"}).Return([]*moira.SubscriptionData{
				{
					Contacts: []string{"testContact"},
					Tags:     []string{"test"},
				},
			}, nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
				User:  defaultLogin,
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

		Convey("Error when deleting a contact, the user has existing emergency contact", func() {
			expected := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  "this contact is being used with emergency contact",
			}

			emergencyContact := datatypes.EmergencyContact{
				ContactID:      contactID,
				HeartbeatTypes: []datatypes.HeartbeatType{datatypes.HeartbeatNotifier},
			}

			mockDb.EXPECT().GetUserSubscriptionIDs(defaultLogin).Return([]string{"test"}, nil).Times(1)
			mockDb.EXPECT().GetEmergencyContact(contactID).Return(emergencyContact, nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
				User:  defaultLogin,
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

		Convey("Internal error when deleting a contact, failed to get emergency contact", func() {
			expected := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  testErr.Error(),
			}

			mockDb.EXPECT().GetUserSubscriptionIDs(defaultLogin).Return([]string{"test"}, nil).Times(1)
			mockDb.EXPECT().GetEmergencyContact(contactID).Return(datatypes.EmergencyContact{}, testErr).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
				User:  defaultLogin,
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

		Convey("Error when deleting a contact, the team has existing subscriptions", func() {
			expected := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  "this contact is being used in following subscriptions:  (tags: test)",
			}

			mockDb.EXPECT().GetTeamSubscriptionIDs(defaultTeamID).Return([]string{"test"}, nil).Times(1)
			mockDb.EXPECT().GetEmergencyContact(contactID).Return(datatypes.EmergencyContact{}, db.ErrNil).Times(1)
			mockDb.EXPECT().GetSubscriptions([]string{"test"}).Return([]*moira.SubscriptionData{
				{
					Contacts: []string{"testContact"},
					Tags:     []string{"test"},
				},
			}, nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
				Team:  defaultTeamID,
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

			mockDb.EXPECT().GetUserSubscriptionIDs(defaultLogin).Return([]string{"test1"}, nil).Times(1)
			mockDb.EXPECT().GetTeamSubscriptionIDs(defaultTeamID).Return([]string{"test2"}, nil).Times(1)
			mockDb.EXPECT().GetEmergencyContact(contactID).Return(datatypes.EmergencyContact{}, db.ErrNil).Times(1)
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
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{
				ID:    contactID,
				Type:  "mail",
				Value: "moira@skbkontur.ru",
				Team:  defaultTeamID,
				User:  defaultLogin,
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

			mockDb.EXPECT().GetEmergencyContact(contactID).Return(datatypes.EmergencyContact{}, db.ErrNil).Times(1)
			mockDb.EXPECT().GetSubscriptions([]string{}).Return([]*moira.SubscriptionData{}, nil).Times(1)
			mockDb.EXPECT().RemoveContact(contactID).Return(testErr).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/contact/"+contactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactKey, moira.ContactData{
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
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactIDKey, contactID))

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
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactIDKey, contactID))

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
