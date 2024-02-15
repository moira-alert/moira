package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/api/dto"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
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

			testRequest := httptest.NewRequest(http.MethodPost, "/tag", bytes.NewBuffer(jsonTags))
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

			testRequest := httptest.NewRequest(http.MethodPost, "/tag", bytes.NewBuffer(jsonTags))
			testRequest.Header.Add("content-type", "application/json")

			createTags(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})
	})
}
