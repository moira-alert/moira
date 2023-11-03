package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/middleware"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDeleteTriggerMetric(t *testing.T) {
	Convey("Delete metric by name", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		Convey("with the correct name", func() {
			mockDb.EXPECT().GetTrigger("triggerID-0000000000001").Return(moira.Trigger{ID: "triggerID-0000000000001"}, nil).Times(1)
			mockDb.EXPECT().AcquireTriggerCheckLock(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			mockDb.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(moira.CheckData{}, nil).Times(1)
			mockDb.EXPECT().DeleteTriggerCheckLock(gomock.Any()).Return(nil).Times(1)
			mockDb.EXPECT().RemovePatternsMetrics(gomock.Any()).Return(nil).Times(1)
			mockDb.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodDelete, "/trigger/triggerID-0000000000001/metrics?name=test", nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "triggerID", "triggerID-0000000000001"))

			deleteTriggerMetric(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)

			So(contents, ShouldEqual, "")
			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("with wrong name", func() {
			testRequest := httptest.NewRequest(http.MethodDelete, "/trigger/triggerID-0000000000001/metrics?name=test%name", nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "triggerID", "triggerID-0000000000001"))
			testRequest.Header.Add("content-type", "application/json")

			deleteTriggerMetric(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := `{"status":"Invalid request","error":"invalid URL escape \"%na\""}
`

			So(contents, ShouldEqual, expected)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})
	})
}
