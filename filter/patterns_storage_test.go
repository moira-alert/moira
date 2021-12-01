package filter

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestProcessIncomingMetric(t *testing.T) {
	testPatterns := []string{
		"cpu.used",
		"plain.metric",
		"seriesByTag(\"name=cpu.used\")",
		"seriesByTag(\"name=tag.metric\", \"tag1=val1\")",
		"seriesByTag(\"name=name.metric\")",
	}

	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Scheduler")

	Convey("Create new pattern storage, GetPatterns returns error, should error", t, func() {
		database.EXPECT().GetPatterns().Return(nil, fmt.Errorf("some error here"))
		metrics := metrics.ConfigureFilterMetrics(metrics.NewDummyRegistry())
		_, err := NewPatternStorage(database, metrics, logger)
		So(err, ShouldBeError, fmt.Errorf("some error here"))
	})

	database.EXPECT().GetPatterns().Return(testPatterns, nil)
	patternsStorage, err := NewPatternStorage(database, metrics.ConfigureFilterMetrics(metrics.NewDummyRegistry()), logger)

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
		patternsStorage.metrics = metrics.ConfigureFilterMetrics(metrics.NewDummyRegistry())
		Convey("For plain metric", func() {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("disk.used 12 1234567890"))
			So(matchedMetrics, ShouldBeNil)
			So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, 0)
		})

		Convey("For tag metric", func() {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("disk.used;tag1=val1 12 1234567890"))
			So(matchedMetrics, ShouldBeNil)
			So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, 0)
		})

		Convey("For plain metric which has the same pattern like name tag in tagged pattern", func() {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("tag.metric 12 1234567890"))
			So(matchedMetrics, ShouldBeNil)
			So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, 0)
		})

		Convey("For tagged metric which body matches plain metric trigger pattern", func() {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("plain.metric;tag1=val1 12 1234567890"))
			So(matchedMetrics, ShouldBeNil)
			So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, 0)
		})

		Convey("For plain metric which matches to tagged pattern which contains only name tag", func() {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("name.metric 12 1234567890"))
			So(matchedMetrics, ShouldBeNil)
			So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, 0)
		})
	})

	Convey("When valid matching metric arrives", t, func() {
		patternsStorage.metrics = metrics.ConfigureFilterMetrics(metrics.NewDummyRegistry())
		Convey("For plain metric", func() {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("plain.metric 12 1234567890"))
			So(matchedMetrics, ShouldNotBeNil)
			So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, 1)
		})
		Convey("For tagged metric", func() {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("tag.metric;tag1=val1 12 1234567890"))
			So(matchedMetrics, ShouldNotBeNil)
			So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, 1)
		})

		Convey("For tagged metric which matches to tagged pattern which contains only name tag", func() {
			matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("name.metric;tag1=val1 12 1234567890"))
			So(matchedMetrics, ShouldNotBeNil)
			So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, 1)
			So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, 1)
		})
	})

	Convey("When ten valid metrics arrive match timer should be updated", t, func() {
		patternsStorage.metrics = metrics.ConfigureFilterMetrics(metrics.NewDummyRegistry())
		for i := 0; i < 10; i++ {
			patternsStorage.ProcessIncomingMetric([]byte("cpu.used 12 1234567890"))
		}
		So(patternsStorage.metrics.MatchingTimer.Count(), ShouldEqual, 1)
	})

	mockCtrl.Finish()
}
