package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"github.com/moira-alert/moira/notifier/selfstate"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
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
			AdminList: map[string]struct{}{adminLogin: {}},
		}}
		webConfig := &api.WebConfig{
			SupportEmail: "test",
			Contacts:     []api.WebContact{},
		}
		handler := NewHandler(mockDb, logger, nil, config, nil, webConfig, nil)

		Convey("Admin tries to set notifier state", func() {
			mockDb.EXPECT().SetNotifierState(moira.SelfStateActorManual, "OK").Return(nil).Times(1)

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

func TestGetSysSubscriptionsWithAuth(t *testing.T) {
	Convey("Authorization enabled", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)
		database = mockDb

		logger, _ := zerolog_adapter.GetLogger("Test")

		adminLogin := "admin_login"
		config := &api.Config{
			Authorization: api.Authorization{
				Enabled:   true,
				AdminList: map[string]struct{}{adminLogin: {}},
			},
		}
		webConfig := &api.WebConfig{
			SupportEmail: "test",
			Contacts:     []api.WebContact{},
		}
		selfchecksConfig := &selfstate.ChecksConfig{
			Database: selfstate.HeartbeatConfig{
				SystemTags: []string{"database-tag"},
			},
			RemoteChecker: selfstate.HeartbeatConfig{
				SystemTags: []string{"remote-checker-tag"},
			},
			LocalChecker: selfstate.HeartbeatConfig{
				SystemTags: []string{"local-checker-tag"},
			},
		}
		handler := NewHandler(mockDb, logger, nil, config, nil, webConfig, selfchecksConfig)

		Convey("Admin tries to get system subscriptions", func() {
			Convey("Without filter", func() {
				mockDb.EXPECT().GetTagsSubscriptions(gomock.Any()).Return(nil, nil)

				testRequest := httptest.NewRequest(http.MethodGet, "/api/health/system-subscriptions", bytes.NewReader([]byte{}))
				testRequest.Header.Set("x-webauth-user", adminLogin)

				handler.ServeHTTP(responseWriter, testRequest)

				response := responseWriter.Result()
				defer response.Body.Close()
				So(response.StatusCode, ShouldEqual, http.StatusOK)
			})
			Convey("With filter", func() {
				mockDb.EXPECT().GetTagsSubscriptions([]string{"remote-checker-tag", "local-checker-tag"}).Return(nil, nil)

				testRequest := httptest.NewRequest(http.MethodGet, "/api/health/system-subscriptions?tag=remote-checker-tag&tag=local-checker-tag", bytes.NewReader([]byte{}))
				testRequest.Header.Set("x-webauth-user", adminLogin)

				handler.ServeHTTP(responseWriter, testRequest)

				response := responseWriter.Result()
				defer response.Body.Close()
				So(response.StatusCode, ShouldEqual, http.StatusOK)
			})
		})

		Convey("Non-admin tries to get system subscriptions", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/api/health/system-subscriptions", bytes.NewReader([]byte{}))
			testRequest.Header.Set("x-webauth-user", "non-admin")

			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			So(response.StatusCode, ShouldEqual, http.StatusForbidden)
		})
	})
}
