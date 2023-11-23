package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
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
		expectedConfig := []byte("Expected config")
		handler := NewHandler(mockDb, logger, nil, config, nil, expectedConfig)

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
			actual, _ := io.ReadAll(response.Body)

			So(response.StatusCode, ShouldEqual, http.StatusOK)
			So(actual, ShouldResemble, expectedConfig)
		})
	})
}
