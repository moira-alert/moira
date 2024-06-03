package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/middleware"
	metricSource "github.com/moira-alert/moira/metric_source"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRenderTrigger(t *testing.T) {
	Convey("Checking the correctness of parameters", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
		remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
		sourceProvider := metricSource.CreateTestMetricSourceProvider(localSource, remoteSource, nil)

		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		Convey("with the wrong realtime parameter", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/trigger/triggerID-0000000000001/render?realtime=test", nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "triggerID", "triggerID-0000000000001"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "metricSourceProvider", sourceProvider))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "target", "t1"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "from", "-1hour"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "to", "now"))

			renderTrigger(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := `{"status":"Invalid request","error":"invalid realtime param: strconv.ParseBool: parsing \"test\": invalid syntax"}
`

			So(contents, ShouldEqual, expected)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("with the wrong timezone parameter", func() {
			mockDb.EXPECT().GetTrigger("triggerID-0000000000001").Return(moira.Trigger{
				ID:            "triggerID-0000000000001",
				Targets:       []string{"t1"},
				TriggerSource: moira.GraphiteLocal,
				ClusterId:     moira.DefaultCluster,
			}, nil).Times(1)
			fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).Times(1)
			localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).Times(1)

			database = mockDb

			testRequest := httptest.NewRequest(http.MethodGet, "/trigger/triggerID-0000000000001/render?timezone=test", nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "triggerID", "triggerID-0000000000001"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "metricSourceProvider", sourceProvider))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "target", "t1"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "from", "-1hour"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "to", "now"))

			renderTrigger(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := `{"status":"Internal Server Error","error":"failed to load test timezone: unknown time zone test"}
`

			So(contents, ShouldEqual, expected)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})

		Convey("without points for render", func() {
			mockDb.EXPECT().GetTrigger("triggerID-0000000000001").Return(moira.Trigger{
				ID:            "triggerID-0000000000001",
				Targets:       []string{"t1"},
				TriggerSource: moira.GraphiteLocal,
				ClusterId:     moira.DefaultCluster,
			}, nil).Times(1)
			fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).Times(1)
			localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).Times(1)

			database = mockDb

			testRequest := httptest.NewRequest(http.MethodGet, "/trigger/triggerID-0000000000001/render", nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "triggerID", "triggerID-0000000000001"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "metricSourceProvider", sourceProvider))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "target", "t1"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "from", "-1hour"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "to", "now"))

			renderTrigger(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := `{"status":"Internal Server Error","error":"no points found to render trigger: triggerID-0000000000001"}
`

			So(contents, ShouldEqual, expected)
			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})

		Convey("with the wrong query string", func() {
			testRequest := httptest.NewRequest(http.MethodGet, "/trigger/triggerID-0000000000001/render?realtime=test%rt", nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "triggerID", "triggerID-0000000000001"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "metricSourceProvider", sourceProvider))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "target", "t1"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "from", "-1hour"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "to", "now"))

			renderTrigger(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()
			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := `{"status":"Invalid request","error":"failed to parse query string: invalid URL escape \"%rt\""}
`

			So(contents, ShouldEqual, expected)
			So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
		})
	})
}
