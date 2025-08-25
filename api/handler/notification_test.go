package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/logging/zerolog_adapter"
	metricsource "github.com/moira-alert/moira/metric_source"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetNotifications(t *testing.T) {
	t.Run("test get notifications url parameters", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		t.Run("with the correct parameters", func(t *testing.T) {
			parameters := []string{"start=0&end=100", "start=0", "end=100", "", "start=test&end=100", "start=0&end=test"}
			for _, param := range parameters {
				mockDb.EXPECT().GetNotifications(gomock.Any(), gomock.Any()).Return([]*moira.ScheduledNotification{}, int64(0), nil).Times(1)
				database = mockDb

				testRequest := httptest.NewRequest(http.MethodGet, "/notifications?"+param, nil)

				getNotification(responseWriter, testRequest)

				response := responseWriter.Result()
				defer response.Body.Close()

				assert.Equal(t, http.StatusOK, response.StatusCode)
			}
		})

		t.Run("with the wrong url query string", func(t *testing.T) {
			testRequest := httptest.NewRequest(http.MethodGet, "/notifications?start=test%&end=100", nil)

			getNotification(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := `{"status":"Invalid request","error":"invalid URL escape \"%\""}
`

			assert.Equal(t, expected, contents)
			assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		})
	})
}

func TestDeleteNotifications(t *testing.T) {
	t.Run("test delete notifications url parameters", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		t.Run("with the empty id parameter", func(t *testing.T) {
			testRequest := httptest.NewRequest(http.MethodDelete, `/notifications`, nil)

			deleteNotification(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := `{"status":"Invalid request","error":"notification id can not be empty"}
`

			assert.Equal(t, expected, contents)
			assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		})

		t.Run("with the correct id", func(t *testing.T) {
			mockDb.EXPECT().RemoveNotification(gomock.Any()).Return(int64(0), nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, `/notifications?id=test`, nil)

			deleteNotification(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := `{"result":0}
`

			assert.Equal(t, expected, contents)
			assert.Equal(t, http.StatusOK, response.StatusCode)
		})

		t.Run("with the wrong url query string", func(t *testing.T) {
			testRequest := httptest.NewRequest(http.MethodDelete, `/notifications?id=test%&`, nil)

			deleteNotification(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := `{"status":"Invalid request","error":"invalid URL escape \"%\""}
`

			assert.Equal(t, expected, contents)
			assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		})
	})
}

func TestDeleteNotificationsFiltered(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	logger, _ := zerolog_adapter.GetLogger("Test")
	mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)
	mockSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	provider := metricsource.CreateTestMetricSourceProvider(mockSource, mockSource, mockSource)
	handler := NewHandler(mockDb, logger, nil, &api.Config{}, provider, nil, nil)

	now := time.Now()
	start := now.Unix()
	end := now.Add(-time.Minute * 10).Unix()

	tag1 := "tag1"
	tag2 := "tag2"

	t.Run("with time parameters", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()

		mockDb.EXPECT().RemoveFilteredNotifications(start, end, gomock.Any(), []moira.ClusterKey{})
		url := fmt.Sprintf("/api/notification/filtered?start=%d&end=%d", start, end)
		testRequest := httptest.NewRequest(http.MethodDelete, url, bytes.NewReader([]byte{}))

		handler.ServeHTTP(responseWriter, testRequest)

		response := responseWriter.Result()
		defer response.Body.Close()

		require.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("with time and single tag parameter", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()

		mockDb.EXPECT().RemoveFilteredNotifications(start, end, []string{tag1}, []moira.ClusterKey{})
		url := fmt.Sprintf("/api/notification/filtered?start=%d&end=%d&ignoredTags[0]=%s", start, end, tag1)
		testRequest := httptest.NewRequest(http.MethodDelete, url, bytes.NewReader([]byte{}))

		handler.ServeHTTP(responseWriter, testRequest)

		response := responseWriter.Result()
		defer response.Body.Close()

		require.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("with time and several tag parameters", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()

		mockDb.EXPECT().RemoveFilteredNotifications(start, end, []string{tag1, tag2}, []moira.ClusterKey{})
		url := fmt.Sprintf("/api/notification/filtered?start=%d&end=%d&ignoredTags[0]=%s&ignoredTags[1]=%s", start, end, tag1, tag2)
		testRequest := httptest.NewRequest(http.MethodDelete, url, bytes.NewReader([]byte{}))

		handler.ServeHTTP(responseWriter, testRequest)

		response := responseWriter.Result()
		defer response.Body.Close()

		require.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("with time and cluster list", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()

		mockDb.EXPECT().RemoveFilteredNotifications(start, end, gomock.Any(), []moira.ClusterKey{moira.DefaultLocalCluster})
		url := fmt.Sprintf("/api/notification/filtered?start=%d&end=%d&clusterKeys[0]=%s", start, end, moira.DefaultLocalCluster.String())
		testRequest := httptest.NewRequest(http.MethodDelete, url, bytes.NewReader([]byte{}))

		handler.ServeHTTP(responseWriter, testRequest)

		response := responseWriter.Result()
		defer response.Body.Close()

		require.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("with time and unknown cluster", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		url := fmt.Sprintf("/api/notification/filtered?start=%d&end=%d&clusterKeys[0]=graphite_local.unknown", start, end)
		testRequest := httptest.NewRequest(http.MethodDelete, url, bytes.NewReader([]byte{}))

		handler.ServeHTTP(responseWriter, testRequest)

		response := responseWriter.Result()
		defer response.Body.Close()

		require.Equal(t, http.StatusBadRequest, response.StatusCode)
	})
}
