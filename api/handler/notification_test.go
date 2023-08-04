package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetNotifications(t *testing.T) {
	Convey("test get notifications url parameters", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		Convey("with the correct parameters", func() {
			parameters := []string{"start=0&end=100", "start=0", "end=100", "", "start=test&end=100", "start=0&end=test"}
			for _, param := range parameters {
				mockDb.EXPECT().GetNotifications(gomock.Any(), gomock.Any()).Return([]*moira.ScheduledNotification{}, int64(0), nil)
				database = mockDb

				testRequest := httptest.NewRequest(http.MethodGet, "/notifications?"+param, nil)

				getNotification(responseWriter, testRequest)

				response := responseWriter.Result()
				defer response.Body.Close()

				So(response.StatusCode, ShouldEqual, http.StatusOK)
			}
		})

		Convey("with the wrong url query string", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/notifications?start=test%&end=100", nil)

			getNotification(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := `{"status":"Invalid request","error":"invalid URL escape \"%\""}
`

			So(contents, ShouldEqual, expected)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})
	})
}

func TestDeleteNotifications(t *testing.T) {
	Convey("test delete notifications url parameters", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		Convey("with the empty id parameter", func() {
			testRequest := httptest.NewRequest(http.MethodDelete, `/notifications`, nil)

			deleteNotification(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := `{"status":"Invalid request","error":"notification id can not be empty"}
`

			So(contents, ShouldEqual, expected)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("with the correct id", func() {
			mockDb.EXPECT().RemoveNotification(gomock.Any()).Return(int64(0), nil)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, `/notifications?id=test`, nil)

			deleteNotification(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := `{"result":0}
`

			So(contents, ShouldEqual, expected)
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("with the wrong url query string", func() {
			testRequest := httptest.NewRequest(http.MethodDelete, `/notifications?id=test%&`, nil)

			deleteNotification(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := `{"status":"Invalid request","error":"invalid URL escape \"%\""}
`

			So(contents, ShouldEqual, expected)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})
	})
}
