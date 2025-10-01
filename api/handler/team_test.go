package handler

import (
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
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"

	. "github.com/smartystreets/goconvey/convey"
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
	Convey("Test searching teams", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
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

		Convey("when everything ok returns ok", func() {
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

			So(response.StatusCode, ShouldEqual, http.StatusOK)

			content, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)

			var gotDTO dto.TeamsList

			err = json.Unmarshal(content, &gotDTO)
			So(err, ShouldBeNil)
			So(gotDTO, ShouldResemble, expectedDTO)
		})

		Convey("when db returns error returns internal server error", func() {
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

			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)

			content, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)

			var gotDTO errorResponse

			err = json.Unmarshal(content, &gotDTO)
			So(err, ShouldBeNil)
			So(gotDTO, ShouldResemble, expectedDTO)
		})
	})
}
