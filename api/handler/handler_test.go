package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestReadonlyMode(t *testing.T) {
	Convey("Test readonly mode enabled", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)
		database = mockDb

		logger, _ := zerolog_adapter.GetLogger("Test")
		config := &api.Config{Flags: api.FeatureFlags{IsReadonlyEnabled: true}}
		webConfig := &api.WebConfig{
			SupportEmail: "test",
			Contacts:     []api.WebContact{},
		}
		handler := NewHandler(mockDb, logger, nil, config, nil, webConfig)

		Convey("Get notifier health", func() {
			mockDb.EXPECT().GetNotifierState().Return("OK", nil).Times(1)

			expected := &dto.NotifierState{
				State: "OK",
			}

			testRequest := httptest.NewRequest(http.MethodGet, "/api/health/notifier", nil)

			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			content, _ := io.ReadAll(response.Body)
			actual := &dto.NotifierState{}
			err := json.Unmarshal(content, actual)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, expected)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Put notifier health", func() {
			mockDb.EXPECT().SetNotifierState("OK").Return(nil).Times(1)

			state := &dto.NotifierState{
				State: "OK",
			}

			stateBytes, err := json.Marshal(state)
			So(err, ShouldBeNil)

			testRequest := httptest.NewRequest(http.MethodPut, "/api/health/notifier", bytes.NewReader(stateBytes))

			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Put new trigger", func() {
			trigger := &dto.Trigger{}
			triggerBytes, err := json.Marshal(trigger)
			So(err, ShouldBeNil)

			testRequest := httptest.NewRequest(http.MethodPut, "/api/trigger", bytes.NewReader(triggerBytes))

			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			So(response.StatusCode, ShouldEqual, http.StatusForbidden)
		})

		Convey("Get contact", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/api/config", nil)

			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			actual, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)
			actualStr := strings.TrimSpace(string(actual))

			expected, err := json.Marshal(webConfig)
			So(err, ShouldBeNil)
			expectedStr := strings.TrimSpace(string(expected))

			So(response.StatusCode, ShouldEqual, http.StatusOK)
			So(actualStr, ShouldResemble, expectedStr)
		})
	})
}

func TestAdminOnly(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)
	database = mockDb

	logger, _ := zerolog_adapter.GetLogger("Test")

	adminLogin := "admin_login"
	userLogin := "user_login"
	config := &api.Config{Authorization: api.Authorization{Enabled: true, AdminList: map[string]struct{}{adminLogin: {}}}}
	webConfig := &api.WebConfig{
		SupportEmail: "test",
		Contacts:     []api.WebContact{},
	}
	handler := NewHandler(mockDb, logger, nil, config, nil, webConfig)

	Convey("Get all contacts", t, func() {
		Convey("For non-admin", func() {
			trigger := &dto.Trigger{}
			triggerBytes, err := json.Marshal(trigger)
			So(err, ShouldBeNil)

			testRequest := httptest.NewRequest(http.MethodGet, "/api/contact", bytes.NewReader(triggerBytes))
			testRequest.Header.Add("x-webauth-user", userLogin)

			responseWriter := httptest.NewRecorder()
			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			So(response.StatusCode, ShouldEqual, http.StatusForbidden)
		})

		Convey("For admin", func() {
			trigger := &dto.Trigger{}
			triggerBytes, err := json.Marshal(trigger)
			So(err, ShouldBeNil)

			mockDb.EXPECT().GetAllContacts().Return([]*moira.ContactData{}, nil)

			testRequest := httptest.NewRequest(http.MethodGet, "/api/contact", bytes.NewReader(triggerBytes))
			testRequest.Header.Add("x-webauth-user", adminLogin)

			responseWriter := httptest.NewRecorder()
			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})
	})

	Convey("Get tag stats", t, func() {
		Convey("For non-admin", func() {
			trigger := &dto.Trigger{}
			triggerBytes, err := json.Marshal(trigger)
			So(err, ShouldBeNil)

			testRequest := httptest.NewRequest(http.MethodGet, "/api/tag/stats", bytes.NewReader(triggerBytes))
			testRequest.Header.Add("x-webauth-user", userLogin)

			responseWriter := httptest.NewRecorder()
			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			So(response.StatusCode, ShouldEqual, http.StatusForbidden)
		})

		Convey("For admin", func() {
			trigger := &dto.Trigger{}
			triggerBytes, err := json.Marshal(trigger)
			So(err, ShouldBeNil)

			mockDb.EXPECT().GetTagNames().Return([]string{"tag_1"}, nil)
			mockDb.EXPECT().GetTagsSubscriptions([]string{"tag_1"}).Return([]*moira.SubscriptionData{}, nil)
			mockDb.EXPECT().GetTagTriggerIDs("tag_1").Return([]string{"tag_1_trigger_id"}, nil)

			testRequest := httptest.NewRequest(http.MethodGet, "/api/tag/stats", bytes.NewReader(triggerBytes))
			testRequest.Header.Add("x-webauth-user", adminLogin)

			responseWriter := httptest.NewRecorder()
			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})
	})
}
