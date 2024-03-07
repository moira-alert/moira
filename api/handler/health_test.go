package handler

import (
	"bytes"
	"encoding/json"
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

func TestSetHealthWithAuth(t *testing.T) {
	Convey("Authorization enabled", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)
		database = mockDb

		logger, _ := zerolog_adapter.GetLogger("Test")

		adminLogin := "admin_login"
		config := &api.Config{Authorization: api.Authorization{
			Enabled:   true,
			AdminList: []string{adminLogin},
		}}
		webConfig := &api.WebConfig{
			SupportEmail: "test",
			Contacts:     []api.WebContact{},
		}
		handler := NewHandler(mockDb, logger, nil, config, nil, webConfig)

		Convey("Admin tries to set notifier state", func() {
			mockDb.EXPECT().SetNotifierState("OK").Return(nil).Times(1)

			state := &dto.NotifierState{
				State: "OK",
			}

			stateBytes, err := json.Marshal(state)
			So(err, ShouldBeNil)

			testRequest := httptest.NewRequest(http.MethodPut, "/api/health/notifier", bytes.NewReader(stateBytes))
			testRequest.Header.Set("x-webauth-user", adminLogin)

			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Non-admin tries to set notifier state", func() {
			state := &dto.NotifierState{
				State: "OK",
			}

			stateBytes, err := json.Marshal(state)
			So(err, ShouldBeNil)

			testRequest := httptest.NewRequest(http.MethodPut, "/api/health/notifier", bytes.NewReader(stateBytes))
			testRequest.Header.Set("x-webauth-user", "non-admin")

			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusForbidden)
		})
	})
}
