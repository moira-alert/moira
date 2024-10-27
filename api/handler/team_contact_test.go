package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	db "github.com/moira-alert/moira/database"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"go.uber.org/mock/gomock"

	. "github.com/smartystreets/goconvey/convey"
)

const testTeamIDKey = "teamID"

func TestCreateNewTeamContact(t *testing.T) {
	Convey("Test create new team contact", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		team := defaultTeamID
		targetRoute := fmt.Sprintf("/api/teams/%s/contacts", team)
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
			User:   "",
			TeamID: team,
		}

		Convey("Correctly create new team contact with the given id", func() {
			jsonContact, err := json.Marshal(newContactDto)
			So(err, ShouldBeNil)

			mockDb.EXPECT().GetContact(defaultContact).Return(moira.ContactData{}, db.ErrNil).Times(1)
			mockDb.EXPECT().SaveContact(&moira.ContactData{
				ID:    newContactDto.ID,
				Name:  newContactDto.Name,
				Type:  newContactDto.Type,
				Value: newContactDto.Value,
				User:  "",
				Team:  newContactDto.TeamID,
			}).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPost, targetRoute, bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testTeamIDKey, team))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
			testRequest.Header.Add("content-type", "application/json")

			createNewTeamContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)

			actual := &dto.Contact{}
			err = json.Unmarshal(contentBytes, actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, newContactDto)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Correctly create team contact without given id", func() {
			newContactDto.ID = ""
			defer func() {
				newContactDto.ID = defaultContact
			}()

			jsonContact, err := json.Marshal(newContactDto)
			So(err, ShouldBeNil)

			mockDb.EXPECT().SaveContact(gomock.Any()).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPost, targetRoute, bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testTeamIDKey, team))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
			testRequest.Header.Add("content-type", "application/json")

			createNewTeamContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)

			actual := &dto.Contact{}
			err = json.Unmarshal(contentBytes, actual)
			So(err, ShouldBeNil)

			So(actual.TeamID, ShouldEqual, newContactDto.TeamID)
			So(actual.Type, ShouldEqual, newContactDto.Type)
			So(actual.User, ShouldEqual, newContactDto.User)
			So(actual.Value, ShouldEqual, newContactDto.Value)
			So(actual.Name, ShouldEqual, newContactDto.Name)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Trying to create a new team contact with the id of an existing contact", func() {
			expected := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  "contact with this ID already exists",
			}
			jsonContact, err := json.Marshal(newContactDto)
			So(err, ShouldBeNil)

			mockDb.EXPECT().GetContact(newContactDto.ID).Return(moira.ContactData{
				ID:    newContactDto.ID,
				Type:  newContactDto.Type,
				Name:  newContactDto.Name,
				Value: newContactDto.Value,
				User:  newContactDto.User,
				Team:  newContactDto.TeamID,
			}, nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPost, targetRoute, bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testTeamIDKey, team))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
			testRequest.Header.Add("content-type", "application/json")

			createNewTeamContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)

			actual := &api.ErrorResponse{}
			err = json.Unmarshal(contentBytes, actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Internal error when trying to create a new team contact with id", func() {
			expected := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  testErr.Error(),
			}
			jsonContact, err := json.Marshal(newContactDto)
			So(err, ShouldBeNil)

			mockDb.EXPECT().GetContact(newContactDto.ID).Return(moira.ContactData{}, testErr).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPost, targetRoute, bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testTeamIDKey, team))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
			testRequest.Header.Add("content-type", "application/json")

			createNewTeamContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)

			actual := &api.ErrorResponse{}
			err = json.Unmarshal(contentBytes, actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})

		Convey("Invalid request when trying to create a new team contact with invalid value", func() {
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

			testRequest := httptest.NewRequest(http.MethodPost, targetRoute, bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testTeamIDKey, team))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
			testRequest.Header.Add("content-type", "application/json")

			createNewTeamContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)

			actual := &api.ErrorResponse{}
			err = json.Unmarshal(contentBytes, actual)
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

		Convey("Trying to create a team contact when both userLogin and teamID specified", func() {
			newContactDto.User = defaultLogin
			defer func() {
				newContactDto.User = ""
			}()

			expected := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  "contact cannot have both the user field and the team_id field filled in",
			}
			jsonContact, err := json.Marshal(newContactDto)
			So(err, ShouldBeNil)

			testRequest := httptest.NewRequest(http.MethodPost, targetRoute, bytes.NewBuffer(jsonContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testTeamIDKey, team))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testContactsTemplateKey, contactsTemplate))
			testRequest.Header.Add("content-type", "application/json")

			createNewTeamContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)

			actual := &api.ErrorResponse{}
			err = json.Unmarshal(contentBytes, actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})
	})
}
