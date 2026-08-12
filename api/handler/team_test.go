package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	db "github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/logging/zerolog_adapter"
	metricsource "github.com/moira-alert/moira/metric_source"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"
)

func fillContextForTestSearchTeams(ctx context.Context, testPage, testSize int64, searchText *regexp.Regexp, sort api.SortOrder) context.Context {
	ctx = middleware.SetContextValueForTest(ctx, "page", testPage)
	ctx = middleware.SetContextValueForTest(ctx, "size", testSize)
	ctx = middleware.SetContextValueForTest(ctx, "searchText", searchText)
	ctx = middleware.SetContextValueForTest(ctx, "sort", sort)

	return ctx
}

func Test_searchTeams(t *testing.T) {
	t.Run("Test searching teams", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)
		database = mockDb

		var (
			defaultTestPage       int64 = getAllTeamsDefaultPage
			defaultTestSize       int64 = getAllTeamsDefaultSize
			defaultTestSearchText       = regexp.MustCompile(getAllTeamsDefaultRegexTemplate)
			defaultTestSortOrder        = api.NoSortOrder
		)

		testTeamsCount := 7
		testTeams := make([]moira.Team, 0, testTeamsCount)

		for i := 0; i < testTeamsCount; i++ {
			iStr := strconv.FormatInt(int64(i), 10)

			testTeams = append(testTeams, moira.Team{
				ID:   "team-" + iStr,
				Name: "Test team " + iStr,
			})
		}

		t.Run("when everything ok returns ok", func(t *testing.T) {
			responseWriter := httptest.NewRecorder()

			mockDb.EXPECT().GetAllTeams().Return(testTeams, nil)

			testRequest := httptest.NewRequest(http.MethodGet, "/api/teams/all", nil)

			testRequest = testRequest.WithContext(
				fillContextForTestSearchTeams(
					testRequest.Context(),
					defaultTestPage,
					defaultTestSize,
					defaultTestSearchText,
					defaultTestSortOrder))
			testRequest.Header.Add("content-type", "application/json")

			total := int64(len(testTeams))

			expectedDTO := dto.NewTeamsList(testTeams)
			expectedDTO.Page = defaultTestPage
			expectedDTO.Size = defaultTestSize
			expectedDTO.Total = total

			searchTeams(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			require.Equal(t, http.StatusOK, response.StatusCode)

			content, err := io.ReadAll(response.Body)
			require.NoError(t, err)

			var gotDTO dto.TeamsList

			err = json.Unmarshal(content, &gotDTO)
			require.NoError(t, err)
			require.Equal(t, expectedDTO, gotDTO)
		})

		t.Run("when db returns error returns internal server error", func(t *testing.T) {
			responseWriter := httptest.NewRecorder()
			dbErr := errors.New("some error from db")

			mockDb.EXPECT().GetAllTeams().Return(nil, dbErr)

			testRequest := httptest.NewRequest(http.MethodGet, "/api/teams/all", nil)

			testRequest = testRequest.WithContext(
				fillContextForTestSearchTeams(
					testRequest.Context(),
					defaultTestPage,
					defaultTestSize,
					defaultTestSearchText,
					defaultTestSortOrder))
			testRequest.Header.Add("content-type", "application/json")

			type errorResponse struct {
				StatusText string `json:"status" binding:"required"`
				ErrorText  string `json:"error,omitempty"`
			}

			expectedErrResponseFromController := api.ErrorInternalServer(fmt.Errorf("cannot get teams from database: %w", dbErr))
			expectedDTO := errorResponse{
				StatusText: expectedErrResponseFromController.StatusText,
				ErrorText:  expectedErrResponseFromController.ErrorText,
			}

			searchTeams(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			require.Equal(t, http.StatusInternalServerError, response.StatusCode)

			content, err := io.ReadAll(response.Body)
			require.NoError(t, err)

			var gotDTO errorResponse

			err = json.Unmarshal(content, &gotDTO)
			require.NoError(t, err)
			require.Equal(t, expectedDTO, gotDTO)
		})
	})
}

func TestAdminOnlyTeamEditingFeatureFlag(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	logger, _ := zerolog_adapter.GetLogger("Test")
	mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)
	mockSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	provider := metricsource.CreateTestMetricSourceProvider(mockSource, mockSource, mockSource)

	handlerNoAuth := NewHandler(mockDb, logger, nil, &api.Config{Limits: api.GetTestLimitsConfig()}, provider, nil, nil)

	handlerNoFf := NewHandler(mockDb, logger, nil, &api.Config{
		Limits: api.GetTestLimitsConfig(),
		Authorization: api.Authorization{
			Enabled: true,
		},
	}, provider, nil, nil)

	adminLogin := "superman"
	nonAdminLogin := "batman"
	handlerWithFf := NewHandler(mockDb, logger, nil, &api.Config{
		Limits: api.GetTestLimitsConfig(),
		Authorization: api.Authorization{
			Enabled:   true,
			AdminList: map[string]struct{}{adminLogin: {}},
			FeatureFlags: api.AuthorizationFeatureFlags{
				ForbidNonAdminsCreateTeams: true,
			},
		},
	}, provider, nil, nil)

	t.Run("when auth is disabled, everything is allowed", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()

		mockDb.EXPECT().GetTeam("team1").Return(moira.Team{}, db.ErrNil)
		mockDb.EXPECT().SaveTeam("team1", gomock.Any()).Return(nil)
		mockDb.EXPECT().GetUserTeams("anonymous").Return([]string{}, nil)
		mockDb.EXPECT().SaveTeamsAndUsers("team1", gomock.Any(), gomock.Any()).Return(nil)

		url := "/api/teams"
		body := `{"id":"team1","name":"Team 1","description":"Team 1 Desc"}`
		testRequest := httptest.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(body)))

		handlerNoAuth.ServeHTTP(responseWriter, testRequest)

		response := responseWriter.Result()
		defer response.Body.Close()

		require.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("when auth is enabled, and feature flag is down, everything is allowed", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()

		mockDb.EXPECT().GetTeam("team1").Return(moira.Team{}, db.ErrNil)
		mockDb.EXPECT().SaveTeam("team1", gomock.Any()).Return(nil)
		mockDb.EXPECT().GetUserTeams("anonymous").Return([]string{}, nil)
		mockDb.EXPECT().SaveTeamsAndUsers("team1", gomock.Any(), gomock.Any()).Return(nil)

		url := "/api/teams"
		body := `{"id":"team1","name":"Team 1","description":"Team 1 Desc"}`
		testRequest := httptest.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(body)))

		handlerNoFf.ServeHTTP(responseWriter, testRequest)

		response := responseWriter.Result()
		defer response.Body.Close()

		require.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("when auth is enabled, and feature flag is up, everything is allowed for admin", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()

		mockDb.EXPECT().GetTeam("team1").Return(moira.Team{}, db.ErrNil)
		mockDb.EXPECT().SaveTeam("team1", gomock.Any()).Return(nil)
		mockDb.EXPECT().GetUserTeams(adminLogin).Return([]string{}, nil)
		mockDb.EXPECT().SaveTeamsAndUsers("team1", gomock.Any(), gomock.Any()).Return(nil)

		url := "/api/teams"
		body := `{"id":"team1","name":"Team 1","description":"Team 1 Desc"}`
		testRequest := httptest.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(body)))
		testRequest.Header.Add("x-webauth-user", adminLogin)

		handlerWithFf.ServeHTTP(responseWriter, testRequest)

		response := responseWriter.Result()
		defer response.Body.Close()

		require.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("when auth is enabled, and feature flag is up, nothing is allowed for non admin", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()

		url := "/api/teams"
		body := `{"id":"team1","name":"Team 1","description":"Team 1 Desc"}`
		testRequest := httptest.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(body)))
		testRequest.Header.Add("x-webauth-user", nonAdminLogin)

		handlerWithFf.ServeHTTP(responseWriter, testRequest)

		response := responseWriter.Result()
		defer response.Body.Close()

		require.Equal(t, http.StatusForbidden, response.StatusCode)
	})
}
