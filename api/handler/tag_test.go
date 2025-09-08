package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	chi_middleware "github.com/go-chi/chi/middleware"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/logging/zerolog_adapter"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	metricSource "github.com/moira-alert/moira/metric_source"
	metricsource "github.com/moira-alert/moira/metric_source"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"github.com/moira-alert/moira/notifier/selfstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	tagRoute      = "/tag/"
	tagStatsRoute = "/tag/stats"
)

func TestCreateTags(t *testing.T) {
	const selfstateChecksContextKey = "selfstateChecks"

	t.Run("Test create tags", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
		remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
		sourceProvider := metricSource.CreateTestMetricSourceProvider(localSource, remoteSource, nil)

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		emptyTags := dto.TagsData{
			TagNames: make([]string, 0),
		}
		tags := dto.TagsData{
			TagNames: []string{"test1", "test2"},
		}

		t.Run("Success with empty tags", func(t *testing.T) {
			jsonTags, err := json.Marshal(emptyTags)
			require.NoError(t, err)

			mockDb.EXPECT().CreateTags(emptyTags.TagNames).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPost, tagRoute, bytes.NewBuffer(jsonTags))
			testRequest.Header.Add("content-type", "application/json")
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "metricSourceProvider", sourceProvider))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), selfstateChecksContextKey, selfstate.ChecksConfig{}))

			createTags(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			require.Equal(t, http.StatusOK, response.StatusCode)
		})

		t.Run("Success with tags", func(t *testing.T) {
			jsonTags, err := json.Marshal(tags)
			require.NoError(t, err)

			mockDb.EXPECT().CreateTags(tags.TagNames).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPost, tagRoute, bytes.NewBuffer(jsonTags))
			testRequest.Header.Add("content-type", "application/json")
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "metricSourceProvider", sourceProvider))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), selfstateChecksContextKey, selfstate.ChecksConfig{}))

			createTags(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			require.Equal(t, http.StatusOK, response.StatusCode)
		})
	})
}

func TestGetAllTags(t *testing.T) {
	t.Run("Test get all tags", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		expectedEmptyTags := &dto.TagsData{
			TagNames: make([]string, 0),
		}
		expectedTags := &dto.TagsData{
			TagNames: []string{"test1", "test2"},
		}

		t.Run("Successfully get empty tags", func(t *testing.T) {
			mockDb.EXPECT().GetTagNames().Return([]string{}, nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodGet, tagRoute, http.NoBody)

			getAllTags(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			actualTags := &dto.TagsData{}
			err := json.Unmarshal([]byte(contents), actualTags)
			require.NoError(t, err)

			require.Equal(t, expectedEmptyTags, actualTags)
			require.Equal(t, http.StatusOK, response.StatusCode)
		})

		t.Run("Successfully get tags", func(t *testing.T) {
			mockDb.EXPECT().GetTagNames().Return([]string{"test1", "test2"}, nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodGet, tagRoute, http.NoBody)

			getAllTags(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			actualTags := &dto.TagsData{}
			err := json.Unmarshal([]byte(contents), actualTags)
			require.NoError(t, err)

			require.Equal(t, expectedTags, actualTags)
			require.Equal(t, http.StatusOK, response.StatusCode)
		})
	})
}

func TestGetAllTagsAndSubscriptions(t *testing.T) {
	t.Run("Test get all tags and subcriptions", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		logger, _ := logging.GetLogger("Test")

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		expectedEmptyTagsAndSubscriptions := &dto.TagsStatistics{
			List: make([]dto.TagStatistics, 0),
		}
		expectedTagsAndSubscriptions := &dto.TagsStatistics{
			List: []dto.TagStatistics{
				{
					TagName:  defaultTag,
					Triggers: []string{"test1", "test2"},
					Subscriptions: []moira.SubscriptionData{
						{
							ID: "test-sub",
						},
					},
				},
			},
		}

		t.Run("Successfully get empty tags and subscriptions", func(t *testing.T) {
			mockDb.EXPECT().GetTagNames().Return([]string{}, nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodGet, tagStatsRoute, http.NoBody)
			apiLogEntry := middleware.NewLogEntry(logger, testRequest)
			testRequest = testRequest.WithContext(context.WithValue(testRequest.Context(), chi_middleware.LogEntryCtxKey, apiLogEntry))

			getAllTagsAndSubscriptions(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			actualTagsStatisctics := &dto.TagsStatistics{}
			err := json.Unmarshal([]byte(contents), actualTagsStatisctics)
			require.NoError(t, err)

			require.Equal(t, expectedEmptyTagsAndSubscriptions, actualTagsStatisctics)
			require.Equal(t, http.StatusOK, response.StatusCode)
		})

		t.Run("Successfully get all tags and subscriptions", func(t *testing.T) {
			mockDb.EXPECT().GetTagNames().Return([]string{defaultTag}, nil).Times(1)
			mockDb.EXPECT().GetTagsSubscriptions([]string{defaultTag}).Return([]*moira.SubscriptionData{
				{
					ID: "test-sub",
				},
			}, nil).Times(1)
			mockDb.EXPECT().GetTagTriggerIDs(defaultTag).Return([]string{"test1", "test2"}, nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodGet, tagStatsRoute, http.NoBody)
			apiLogEntry := middleware.NewLogEntry(logger, testRequest)
			testRequest = testRequest.WithContext(context.WithValue(testRequest.Context(), chi_middleware.LogEntryCtxKey, apiLogEntry))

			getAllTagsAndSubscriptions(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			actualTagsStatistics := &dto.TagsStatistics{}
			err := json.Unmarshal([]byte(contents), actualTagsStatistics)
			require.NoError(t, err)

			require.Equal(t, expectedTagsAndSubscriptions, actualTagsStatistics)
			require.Equal(t, http.StatusOK, response.StatusCode)
		})
	})
}

func TestRemoveTag(t *testing.T) {
	t.Run("Test remove tag", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		deletedTagMsg := &dto.MessageResponse{
			Message: "tag deleted",
		}

		t.Run("Successfully remove tag", func(t *testing.T) {
			mockDb.EXPECT().GetTagTriggerIDs(defaultTag).Return([]string{}, nil).Times(1)
			mockDb.EXPECT().GetTagsSubscriptions([]string{defaultTag}).Return([]*moira.SubscriptionData{}, nil).Times(1)
			mockDb.EXPECT().RemoveTag(defaultTag).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, tagRoute+defaultTag, http.NoBody)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "tag", defaultTag))

			removeTag(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			actualMsg := &dto.MessageResponse{}
			err := json.Unmarshal([]byte(contents), actualMsg)
			require.NoError(t, err)

			require.Equal(t, actualMsg, deletedTagMsg)
			require.Equal(t, http.StatusOK, response.StatusCode)
		})

		t.Run("Failed to remove tag with an existing trigger", func(t *testing.T) {
			mockDb.EXPECT().GetTagTriggerIDs(defaultTag).Return([]string{defaultTag}, nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, tagRoute+defaultTag, http.NoBody)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "tag", defaultTag))

			removeTag(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			tagExistInTriggerErr := `{"status":"Invalid request","error":"this tag is assigned to 1 triggers. Remove tag from triggers first"}
`
			require.Equal(t, contents, tagExistInTriggerErr)
			require.Equal(t, http.StatusBadRequest, response.StatusCode)
		})

		t.Run("Failed to remove tag with an existing subscription", func(t *testing.T) {
			mockDb.EXPECT().GetTagTriggerIDs(defaultTag).Return([]string{}, nil).Times(1)
			mockDb.EXPECT().GetTagsSubscriptions([]string{defaultTag}).Return([]*moira.SubscriptionData{
				{
					ID: "test-sub",
				},
			}, nil).Times(1)

			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, tagRoute+defaultTag, http.NoBody)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "tag", defaultTag))

			removeTag(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			tagExistInSubscriptionErr := `{"status":"Invalid request","error":"this tag is assigned to 1 subscriptions. Remove tag from subscriptions first"}
`
			require.Equal(t, contents, tagExistInSubscriptionErr)
			require.Equal(t, http.StatusBadRequest, response.StatusCode)
		})
	})
}

func TestGetAllSystemTags(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	responseWriter := httptest.NewRecorder()
	logger, _ := zerolog_adapter.GetLogger("Test")
	mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)
	mockSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	provider := metricsource.CreateTestMetricSourceProvider(mockSource, mockSource, mockSource)
	selfstateConfig := selfstate.ChecksConfig{
		Database: selfstate.HeartbeatConfig{
			SystemTags: []string{"moira-database-fatal"},
		},
		Filter: selfstate.HeartbeatConfig{
			SystemTags: []string{"moira-filter-fatal"},
		},
		LocalChecker: selfstate.HeartbeatConfig{
			SystemTags: []string{"moira-local-checker-fatal"},
		},
		RemoteChecker: selfstate.HeartbeatConfig{
			SystemTags: []string{"moira-remote-checker-fatal"},
		},
		Notifier: selfstate.NotifierHeartbeatConfig{
			DefaultTags:     []string{"moira-notifier-fatal"},
			LocalSourceTags: []string{"moira-local-source-fatal"},
			SourceTagPrefix: "moira-source-fatal",
		},
	}
	handler := NewHandler(mockDb, logger, nil, &api.Config{}, provider, nil, &selfstateConfig)

	t.Run("Successfully get system tags", func(t *testing.T) {
		testRequest := httptest.NewRequest(http.MethodGet, "/api/system-tag", bytes.NewReader([]byte{}))
		handler.ServeHTTP(responseWriter, testRequest)

		response := responseWriter.Result()
		defer response.Body.Close()

		require.Equal(t, http.StatusOK, response.StatusCode)

		contentBytes, _ := io.ReadAll(response.Body)
		actual := &dto.TagsData{}
		err := json.Unmarshal(contentBytes, actual)

		require.NoError(t, err)
		require.ElementsMatch(t, actual.TagNames, []string{
			"moira-database-fatal",
			"moira-filter-fatal",
			"moira-local-checker-fatal",
			"moira-remote-checker-fatal",
			"moira-notifier-fatal",
			"moira-local-source-fatal",
			"moira-source-fatal:graphite_local.default",
			"moira-source-fatal:graphite_remote.default",
			"moira-source-fatal:prometheus_remote.default",
		})
	})
}
