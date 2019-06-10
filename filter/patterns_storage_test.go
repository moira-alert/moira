package filter

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/metrics/graphite/go-metrics"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestProcessIncomingMetric(t *testing.T) {
	testPatterns := []string{
		"cpu.used",
		"seriesByTag(\"name=cpu.used\")",
	}

	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Scheduler")

	Convey("Create new pattern storage, GetPatterns returns error, should error", t, func() {
		database.EXPECT().GetPatterns().Return(nil, fmt.Errorf("some error here"))
		metrics := metrics.ConfigureFilterMetrics("test")
		_, err := NewPatternStorage(database, metrics, logger)
		So(err, ShouldBeError, fmt.Errorf("some error here"))
	})

	database.EXPECT().GetPatterns().Return(testPatterns, nil)
	patternsStorage, err := NewPatternStorage(database, metrics.ConfigureFilterMetrics("test"), logger)

	Convey("Create new pattern storage, should no error", t, func() {
		So(err, ShouldBeEmpty)
	})

	Convey("When invalid metric arrives, should be properly counted", t, func() {
		matchedMetrics := patternsStorage.ProcessIncomingMetric(nil)
		So(matchedMetrics, ShouldBeNil)
		So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, 1)
		So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, 0)
		So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, 0)
	})

	Convey("When valid non-matching metric arrives", t, func() {
		patternsStorage.metrics = metrics.ConfigureFilterMetrics("test")
		matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("disk.used 12 1234567890"))
		So(matchedMetrics, ShouldBeNil)
		So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, 1)
		So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, 1)
		So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, 0)
	})

	Convey("When valid matching metric arrives", t, func() {
		patternsStorage.metrics = metrics.ConfigureFilterMetrics("test")
		matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("cpu.used 12 1234567890"))
		So(matchedMetrics, ShouldNotBeNil)
		So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, 1)
		So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, 1)
		So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, 1)
	})

	Convey("When ten valid metrics arrive match timer should be updated", t, func() {
		patternsStorage.metrics = metrics.ConfigureFilterMetrics("test")
		for i := 0; i < 10; i++ {
			patternsStorage.ProcessIncomingMetric([]byte("cpu.used 12 1234567890"))
		}
		So(patternsStorage.metrics.MatchingTimer.Count(), ShouldEqual, 1)
	})

	mockCtrl.Finish()
}
