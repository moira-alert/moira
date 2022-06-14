package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/middleware"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetTrigger(t *testing.T) {
	Convey("Get trigger by id", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		Convey("When success and have empty created_at & updated_at should return null in response", func() {
			throttlingTime := time.Date(2022, time.June, 7, 10, 0, 0, 0, time.UTC)
			mockDb.EXPECT().GetTrigger("triggerID-0000000000001").Return(moira.Trigger{ID: "triggerID-0000000000001"}, nil)
			mockDb.EXPECT().GetTriggerThrottling("triggerID-0000000000001").Return(throttlingTime, throttlingTime)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodGet, "/trigger/triggerID-0000000000001", nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "triggerID", "triggerID-0000000000001"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "populated", false))
			testRequest.Header.Add("content-type", "application/json")

			getTrigger(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := "{\"id\":\"triggerID-0000000000001\",\"name\":\"\",\"targets\":null,\"warn_value\":null,\"error_value\":null,\"trigger_type\":\"\",\"tags\":null,\"expression\":\"\",\"patterns\":null,\"is_remote\":false,\"mute_new_metrics\":false,\"alone_metrics\":null,\"created_at\":null,\"updated_at\":null,\"throttling\":0}\n"
			So(contents, ShouldEqual, expected)
		})

		Convey("When success and have not empty created_at & updated_at should return datetime in response", func() {
			throttlingTime := time.Date(2022, time.June, 7, 10, 0, 0, 0, time.UTC)
			triggerTime := time.Date(2022, time.June, 7, 10, 0, 0, 0, time.UTC).Unix()
			mockDb.EXPECT().GetTrigger("triggerID-0000000000001").
				Return(
					moira.Trigger{
						ID:        "triggerID-0000000000001",
						CreatedAt: &triggerTime,
						UpdatedAt: &triggerTime,
					},
					nil)
			mockDb.EXPECT().GetTriggerThrottling("triggerID-0000000000001").Return(throttlingTime, throttlingTime)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodGet, "/trigger/triggerID-0000000000001", nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "triggerID", "triggerID-0000000000001"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "populated", false))
			testRequest.Header.Add("content-type", "application/json")

			getTrigger(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := "{\"id\":\"triggerID-0000000000001\",\"name\":\"\",\"targets\":null,\"warn_value\":null,\"error_value\":null,\"trigger_type\":\"\",\"tags\":null,\"expression\":\"\",\"patterns\":null,\"is_remote\":false,\"mute_new_metrics\":false,\"alone_metrics\":null,\"created_at\":\"2022-06-07T10:00:00Z\",\"updated_at\":\"2022-06-07T10:00:00Z\",\"throttling\":0}\n"
			So(contents, ShouldEqual, expected)
		})

		Convey("When cannot get trigger should have error in response", func() {
			mockDb.EXPECT().GetTrigger("triggerID-0000000000001").Return(moira.Trigger{}, fmt.Errorf("cannot get trigger"))
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodGet, "/trigger/triggerID-0000000000001", nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "triggerID", "triggerID-0000000000001"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "populated", false))
			testRequest.Header.Add("content-type", "application/json")

			getTrigger(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := "{\"status\":\"Internal Server Error\",\"error\":\"cannot get trigger\"}\n"
			So(contents, ShouldEqual, expected)
		})
	})
}
