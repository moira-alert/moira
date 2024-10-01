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
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	moiradb "github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/datatypes"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

var (
	testContactID  = "test-contact-id"
	testContactID2 = "test-contact-id2"

	testEmergencyContact = datatypes.EmergencyContact{
		ContactID:      testContactID,
		HeartbeatTypes: []datatypes.HeartbeatType{datatypes.HeartbeatNotifierOff},
	}
	testEmergencyContact2 = datatypes.EmergencyContact{
		ContactID:      testContactID2,
		HeartbeatTypes: []datatypes.HeartbeatType{datatypes.HearbeatTypeNotSet},
	}

	login = "testLogin"

	testContact = moira.ContactData{
		ID:   testContactID,
		User: login,
	}
)

func TestGetEmergencyContacts(t *testing.T) {
	Convey("Test getEmergencyContacts", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		Convey("Successfully get emergency contacts", func() {
			mockDb.EXPECT().GetEmergencyContacts().Return([]*datatypes.EmergencyContact{
				&testEmergencyContact,
				&testEmergencyContact2,
			}, nil)
			database = mockDb

			expectedEmergencyContactList := &dto.EmergencyContactList{
				List: []dto.EmergencyContact{
					dto.EmergencyContact(testEmergencyContact),
					dto.EmergencyContact(testEmergencyContact2),
				},
			}

			testRequest := httptest.NewRequest(http.MethodGet, "/emergency-contact", nil)

			getEmergencyContacts(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			actual := &dto.EmergencyContactList{}
			err := json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedEmergencyContactList)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Internal server error from database", func() {
			dbErr := errors.New("get emergency contacts error")
			mockDb.EXPECT().GetEmergencyContacts().Return(nil, dbErr)
			database = mockDb

			expectedErr := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  dbErr.Error(),
			}

			testRequest := httptest.NewRequest(http.MethodGet, "/emergency-contact", nil)

			getEmergencyContacts(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err := json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedErr)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestGetEmergencyContactByID(t *testing.T) {
	Convey("Test getEmergencyContactByID", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		Convey("Successfully get emergency contact by id", func() {
			mockDb.EXPECT().GetEmergencyContact(testContactID).Return(testEmergencyContact, nil)
			database = mockDb

			expectedEmergencyContactDTO := dto.EmergencyContact(testEmergencyContact)
			testRequest := httptest.NewRequest(http.MethodGet, "/emergency-contact/"+testContactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactIDKey, testContactID))

			getEmergencyContactByID(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &dto.EmergencyContact{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, &expectedEmergencyContactDTO)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Not found error from database", func() {
			dbErr := moiradb.ErrNil
			mockDb.EXPECT().GetEmergencyContact(testContactID).Return(datatypes.EmergencyContact{}, dbErr)
			database = mockDb

			expectedErr := &api.ErrorResponse{
				StatusText: "Resource not found",
				ErrorText:  fmt.Sprintf("emergency contact with ID '%s' does not exists", testContactID),
			}

			testRequest := httptest.NewRequest(http.MethodGet, "/emergency-contact/"+testContactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactIDKey, testContactID))

			getEmergencyContactByID(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedErr)
			So(response.StatusCode, ShouldEqual, http.StatusNotFound)
		})

		Convey("Internal error from database", func() {
			dbErr := errors.New("get emergency contact error")
			mockDb.EXPECT().GetEmergencyContact(testContactID).Return(datatypes.EmergencyContact{}, dbErr)
			database = mockDb

			expectedErr := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  dbErr.Error(),
			}

			testRequest := httptest.NewRequest(http.MethodGet, "/emergency-contact/"+testContactID, nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactIDKey, testContactID))

			getEmergencyContactByID(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedErr)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestCreateEmergencyContact(t *testing.T) {
	Convey("Test createEmergencyContact", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		auth := &api.Authorization{
			Enabled:   true,
			AdminList: map[string]struct{}{login: {}},
		}

		Convey("Successfully create emergency contact", func() {
			emergencyContactDTO := dto.EmergencyContact(testEmergencyContact)

			expectedResponse := &dto.SaveEmergencyContactResponse{
				ContactID: testContactID,
			}

			jsonEmergencyContact, err := json.Marshal(emergencyContactDTO)
			So(err, ShouldBeNil)

			mockDb.EXPECT().GetContact(testContactID).Return(testContact, nil)
			mockDb.EXPECT().SaveEmergencyContact(testEmergencyContact).Return(nil)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPost, "/emergency-contact", bytes.NewBuffer(jsonEmergencyContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), LoginKey, login))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), AuthKey, auth))
			testRequest.Header.Add("content-type", "application/json")

			createEmergencyContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &dto.SaveEmergencyContactResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedResponse)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Try to create emergency contact without contact id", func() {
			emergencyContact := datatypes.EmergencyContact{
				HeartbeatTypes: []datatypes.HeartbeatType{datatypes.HeartbeatNotifierOff},
			}
			emergencyContactDTO := dto.EmergencyContact(emergencyContact)

			jsonEmergencyContact, err := json.Marshal(emergencyContactDTO)
			So(err, ShouldBeNil)

			database = mockDb

			expectedErr := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  controller.ErrEmptyEmergencyContactID.Error(),
			}

			testRequest := httptest.NewRequest(http.MethodPost, "/emergency-contact", bytes.NewBuffer(jsonEmergencyContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), LoginKey, login))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), AuthKey, auth))
			testRequest.Header.Add("content-type", "application/json")

			createEmergencyContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedErr)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Try to create emergency contact without heartbeat types", func() {
			emergencyContact := datatypes.EmergencyContact{
				ContactID: testContactID,
			}
			emergencyContactDTO := dto.EmergencyContact(emergencyContact)

			jsonEmergencyContact, err := json.Marshal(emergencyContactDTO)
			So(err, ShouldBeNil)

			database = mockDb

			expectedErr := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  dto.ErrEmptyHeartbeatTypes.Error(),
			}

			testRequest := httptest.NewRequest(http.MethodPost, "/emergency-contact", bytes.NewBuffer(jsonEmergencyContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), LoginKey, login))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), AuthKey, auth))
			testRequest.Header.Add("content-type", "application/json")

			createEmergencyContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedErr)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Try to create emergency contact with invalid heartbeat type", func() {
			emergencyContact := datatypes.EmergencyContact{
				ContactID: testContactID,
				HeartbeatTypes: []datatypes.HeartbeatType{
					"notifier_on",
				},
			}
			emergencyContactDTO := dto.EmergencyContact(emergencyContact)

			jsonEmergencyContact, err := json.Marshal(emergencyContactDTO)
			So(err, ShouldBeNil)

			database = mockDb

			expectedErr := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  "'notifier_on' heartbeat type doesn't exist",
			}

			testRequest := httptest.NewRequest(http.MethodPost, "/emergency-contact", bytes.NewBuffer(jsonEmergencyContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), LoginKey, login))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), AuthKey, auth))
			testRequest.Header.Add("content-type", "application/json")

			createEmergencyContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedErr)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Internal server error with get contact database error", func() {
			emergencyContactDTO := dto.EmergencyContact(testEmergencyContact)

			jsonEmergencyContact, err := json.Marshal(emergencyContactDTO)
			So(err, ShouldBeNil)

			dbErr := errors.New("get contact error")

			mockDb.EXPECT().GetContact(testContactID).Return(moira.ContactData{}, dbErr)
			database = mockDb

			expectedErr := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  dbErr.Error(),
			}

			testRequest := httptest.NewRequest(http.MethodPost, "/emergency-contact", bytes.NewBuffer(jsonEmergencyContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), LoginKey, login))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), AuthKey, auth))
			testRequest.Header.Add("content-type", "application/json")

			createEmergencyContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedErr)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})

		Convey("Internal server error with save emergency contact database error", func() {
			emergencyContactDTO := dto.EmergencyContact(testEmergencyContact)

			jsonEmergencyContact, err := json.Marshal(emergencyContactDTO)
			So(err, ShouldBeNil)

			dbErr := errors.New("save emergency contact error")

			mockDb.EXPECT().GetContact(testContactID).Return(testContact, nil)
			mockDb.EXPECT().SaveEmergencyContact(testEmergencyContact).Return(dbErr)
			database = mockDb

			expectedErr := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  dbErr.Error(),
			}

			testRequest := httptest.NewRequest(http.MethodPost, "/emergency-contact", bytes.NewBuffer(jsonEmergencyContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), LoginKey, login))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), AuthKey, auth))
			testRequest.Header.Add("content-type", "application/json")

			createEmergencyContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedErr)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestUpdateEmergencyContact(t *testing.T) {
	Convey("Test updateEmergencyContact", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		Convey("Successfully update emergency contact", func() {
			emergencyContactDTO := dto.EmergencyContact(testEmergencyContact)

			expectedResponse := &dto.SaveEmergencyContactResponse{
				ContactID: testContactID,
			}

			jsonEmergencyContact, err := json.Marshal(emergencyContactDTO)
			So(err, ShouldBeNil)

			mockDb.EXPECT().SaveEmergencyContact(testEmergencyContact).Return(nil)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPut, "/emergency-contact/"+testContactID, bytes.NewBuffer(jsonEmergencyContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactIDKey, testContactID))
			testRequest.Header.Add("content-type", "application/json")

			updateEmergencyContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &dto.SaveEmergencyContactResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedResponse)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Successfully update emergency contact without contact id in dto", func() {
			emergencyContact := datatypes.EmergencyContact{
				HeartbeatTypes: []datatypes.HeartbeatType{datatypes.HeartbeatNotifierOff},
			}
			emergencyContactDTO := dto.EmergencyContact(emergencyContact)

			expectedResponse := &dto.SaveEmergencyContactResponse{
				ContactID: testContactID,
			}

			jsonEmergencyContact, err := json.Marshal(emergencyContactDTO)
			So(err, ShouldBeNil)

			mockDb.EXPECT().SaveEmergencyContact(testEmergencyContact).Return(nil)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPut, "/emergency-contact/"+testContactID, bytes.NewBuffer(jsonEmergencyContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactIDKey, testContactID))
			testRequest.Header.Add("content-type", "application/json")

			updateEmergencyContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &dto.SaveEmergencyContactResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedResponse)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Invalid Request without heartbeat types in dto", func() {
			emergencyContact := datatypes.EmergencyContact{
				ContactID: testContactID,
			}
			emergencyContactDTO := dto.EmergencyContact(emergencyContact)

			expectedErr := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  dto.ErrEmptyHeartbeatTypes.Error(),
			}
			jsonEmergencyContact, err := json.Marshal(emergencyContactDTO)
			So(err, ShouldBeNil)

			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPut, "/emergency-contact/"+testContactID, bytes.NewBuffer(jsonEmergencyContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactIDKey, testContactID))
			testRequest.Header.Add("content-type", "application/json")

			updateEmergencyContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedErr)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Invalid Request with undefined heartbeat type in dto", func() {
			emergencyContact := datatypes.EmergencyContact{
				ContactID: testContactID,
				HeartbeatTypes: []datatypes.HeartbeatType{
					"notifier_on",
				},
			}
			emergencyContactDTO := dto.EmergencyContact(emergencyContact)

			expectedErr := &api.ErrorResponse{
				StatusText: "Invalid request",
				ErrorText:  "'notifier_on' heartbeat type doesn't exist",
			}
			jsonEmergencyContact, err := json.Marshal(emergencyContactDTO)
			So(err, ShouldBeNil)

			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPut, "/emergency-contact/"+testContactID, bytes.NewBuffer(jsonEmergencyContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactIDKey, testContactID))
			testRequest.Header.Add("content-type", "application/json")

			updateEmergencyContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedErr)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Internal Server Error with database error", func() {
			emergencyContactDTO := dto.EmergencyContact(testEmergencyContact)

			dbErr := errors.New("update emergency contact error")
			expectedErr := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  dbErr.Error(),
			}
			jsonEmergencyContact, err := json.Marshal(emergencyContactDTO)
			So(err, ShouldBeNil)

			mockDb.EXPECT().SaveEmergencyContact(testEmergencyContact).Return(dbErr)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPut, "/emergency-contact/"+testContactID, bytes.NewBuffer(jsonEmergencyContact))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactIDKey, testContactID))
			testRequest.Header.Add("content-type", "application/json")

			updateEmergencyContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedErr)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestRemoveEmergencyContact(t *testing.T) {
	Convey("Test removeEmergencyContact", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		Convey("Successfully remove emergency contact", func() {
			mockDb.EXPECT().RemoveEmergencyContact(testContactID).Return(nil)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPost, "/emergency-contact/"+testContactID, http.NoBody)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactIDKey, testContactID))

			removeEmergencyContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Internal server error with remove emergency contact", func() {
			dbErr := errors.New("remove emergency contact error")
			mockDb.EXPECT().RemoveEmergencyContact(testContactID).Return(dbErr)
			database = mockDb

			expectedErr := &api.ErrorResponse{
				StatusText: "Internal Server Error",
				ErrorText:  dbErr.Error(),
			}

			testRequest := httptest.NewRequest(http.MethodDelete, "/emergency-contact/"+testContactID, http.NoBody)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), ContactIDKey, testContactID))

			removeEmergencyContact(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			contents := string(contentBytes)
			actual := &api.ErrorResponse{}
			err = json.Unmarshal([]byte(contents), actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expectedErr)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})
	})
}
