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
	metricSource "github.com/moira-alert/moira/metric_source"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"github.com/moira-alert/moira/notifier/selfstate"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSetHealthWithAuth(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

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

	t.Run("Admin tries to set notifier state", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()

		mockDb.EXPECT().SetNotifierState(moira.SelfStateActorManual, "OK", gomock.Any()).Return(nil).Times(1)

		state := &dto.NotifierState{
			State: "OK",
		}

		stateBytes, err := json.Marshal(state)
		require.NoError(t, err)

		testRequest := httptest.NewRequest(http.MethodPut, "/api/health/notifier", bytes.NewReader(stateBytes))
		testRequest.Header.Set("x-webauth-user", adminLogin)

		handler.ServeHTTP(responseWriter, testRequest)

		response := responseWriter.Result()
		defer response.Body.Close()

		require.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("Non-admin tries to set notifier state", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		state := &dto.NotifierState{
			State: "OK",
		}

		stateBytes, err := json.Marshal(state)
		require.NoError(t, err)

		testRequest := httptest.NewRequest(http.MethodPut, "/api/health/notifier", bytes.NewReader(stateBytes))
		testRequest.Header.Set("x-webauth-user", "non-admin")

		handler.ServeHTTP(responseWriter, testRequest)

		response := responseWriter.Result()
		defer response.Body.Close()

		require.Equal(t, http.StatusForbidden, response.StatusCode)
	})
}

func TestSetHealthForSourceWithAuth(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

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
	localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	handler := NewHandler(mockDb, logger, nil, config, metricSource.CreateTestMetricSourceProvider(localSource, nil, nil), webConfig, nil)

	t.Run("Admin tries to set notifier state", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()

		mockDb.EXPECT().SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorManual, "OK", gomock.Any()).Return(nil).Times(1)

		state := &dto.NotifierState{
			State: "OK",
		}

		stateBytes, err := json.Marshal(state)
		require.NoError(t, err)

		testRequest := httptest.NewRequest(http.MethodPut, "/api/health/notifier-sources/graphite_local/default", bytes.NewReader(stateBytes))
		testRequest.Header.Set("x-webauth-user", adminLogin)

		handler.ServeHTTP(responseWriter, testRequest)

		response := responseWriter.Result()
		defer response.Body.Close()

		require.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("Non-admin tries to set notifier state", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		state := &dto.NotifierState{
			State: "OK",
		}

		stateBytes, err := json.Marshal(state)
		require.NoError(t, err)

		testRequest := httptest.NewRequest(http.MethodPut, "/api/health/notifier-sources/graphite_local/default", bytes.NewReader(stateBytes))
		testRequest.Header.Set("x-webauth-user", "non-admin")

		handler.ServeHTTP(responseWriter, testRequest)

		response := responseWriter.Result()
		defer response.Body.Close()

		require.Equal(t, http.StatusForbidden, response.StatusCode)
	})
}

func TestGetSysSubscriptionsWithAuth(t *testing.T) {
	t.Run("Authorization enabled", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
		remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
		sourceProvider := metricSource.CreateTestMetricSourceProvider(localSource, remoteSource, nil)

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
		handler := NewHandler(mockDb, logger, nil, config, sourceProvider, webConfig, selfchecksConfig)

		t.Run("Admin tries to get system subscriptions", func(t *testing.T) {
			t.Run("Without filter", func(t *testing.T) {
				responseWriter := httptest.NewRecorder()

				mockDb.EXPECT().GetTagsSubscriptions(gomock.Any()).Return(nil, nil)

				testRequest := httptest.NewRequest(http.MethodGet, "/api/health/system-subscriptions", bytes.NewReader([]byte{}))
				testRequest.Header.Set("x-webauth-user", adminLogin)

				handler.ServeHTTP(responseWriter, testRequest)

				response := responseWriter.Result()
				defer response.Body.Close()

				require.Equal(t, http.StatusOK, response.StatusCode)
			})
			t.Run("With filter", func(t *testing.T) {
				responseWriter := httptest.NewRecorder()

				mockDb.EXPECT().GetTagsSubscriptions([]string{"remote-checker-tag", "local-checker-tag"}).Return(nil, nil)

				testRequest := httptest.NewRequest(http.MethodGet, "/api/health/system-subscriptions?tag=remote-checker-tag&tag=local-checker-tag", bytes.NewReader([]byte{}))
				testRequest.Header.Set("x-webauth-user", adminLogin)

				handler.ServeHTTP(responseWriter, testRequest)

				response := responseWriter.Result()
				defer response.Body.Close()

				require.Equal(t, http.StatusOK, response.StatusCode)
			})
		})

		t.Run("Non-admin tries to get system subscriptions", func(t *testing.T) {
			responseWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest(http.MethodGet, "/api/health/system-subscriptions", bytes.NewReader([]byte{}))
			testRequest.Header.Set("x-webauth-user", "non-admin")

			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			require.Equal(t, http.StatusForbidden, response.StatusCode)
		})
	})
}
