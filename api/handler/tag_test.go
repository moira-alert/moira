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
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

const (
	tagRoute      = "/tag/"
	tagStatsRoute = "/tag/stats"
)

func TestCreateTags(t *testing.T) {
	Convey("Test create tags", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		emptyTags := dto.TagsData{
			TagNames: make([]string, 0),
		}
		tags := dto.TagsData{
			TagNames: []string{"test1", "test2"},
		}

		Convey("Success with empty tags", func() {
			jsonTags, err := json.Marshal(emptyTags)
			So(err, ShouldBeNil)

			mockDb.EXPECT().CreateTags(emptyTags.TagNames).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPost, tagRoute, bytes.NewBuffer(jsonTags))
			testRequest.Header.Add("content-type", "application/json")

			createTags(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Success with tags", func() {
			jsonTags, err := json.Marshal(tags)
			So(err, ShouldBeNil)

			mockDb.EXPECT().CreateTags(tags.TagNames).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodPost, tagRoute, bytes.NewBuffer(jsonTags))
			testRequest.Header.Add("content-type", "application/json")

			createTags(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})
	})
}

func TestGetAllTags(t *testing.T) {
	Convey("Test get all tags", t, func() {
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

		Convey("Successfully get empty tags", func() {
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
			So(err, ShouldBeNil)

			So(actualTags, ShouldResemble, expectedEmptyTags)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Successfully get tags", func() {
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
			So(err, ShouldBeNil)

			So(actualTags, ShouldResemble, expectedTags)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})
	})
}

func TestGetAllTagsAndSubscriptions(t *testing.T) {
	Convey("Test get all tags and subcriptions", t, func() {
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

		Convey("Successfully get empty tags and subscriptions", func() {
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
			So(err, ShouldBeNil)

			So(actualTagsStatisctics, ShouldResemble, expectedEmptyTagsAndSubscriptions)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Successfully get all tags and subscriptions", func() {
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
			So(err, ShouldBeNil)

			So(actualTagsStatistics, ShouldResemble, expectedTagsAndSubscriptions)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})
	})
}

func TestRemoveTag(t *testing.T) {
	Convey("Test remove tag", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		deletedTagMsg := &dto.MessageResponse{
			Message: "tag deleted",
		}

		Convey("Successfully remove tag", func() {
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
			So(err, ShouldBeNil)

			So(actualMsg, ShouldResemble, deletedTagMsg)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Failed to remove tag with an existing trigger", func() {
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
			So(contents, ShouldEqual, tagExistInTriggerErr)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Failed to remove tag with an existing subscription", func() {
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
			So(contents, ShouldEqual, tagExistInSubscriptionErr)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})
	})
}
