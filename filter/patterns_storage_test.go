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
		"Simple.matching.pattern",
		"Star.single.*",
		"Star.*.double.any*",
		"Bracket.{one,two,three}.pattern",
		"Bracket.pr{one,two,three}suf",
		"Complex.matching.pattern",
		"Complex.*.*",
		"Complex.*.",
		"Complex.*{one,two,three}suf*.pattern",
		"Question.?at_begin",
		"Question.at_the_end?",
	}

	nonMatchingMetrics := []string{
		"Simple.notmatching.pattern",
		"Star.nothing",
		"Bracket.one.nothing",
		"Bracket.nothing.pattern",
		"Complex.prefixonesuffix",
	}

	matchingMetrics := []string{
		"Simple.matching.pattern",
		"Star.single.anything",
		"Star.anything.double.anything",
		"Bracket.one.pattern",
		"Bracket.two.pattern",
		"Bracket.three.pattern",
		"Bracket.pronesuf",
		"Bracket.prtwosuf",
		"Bracket.prthreesuf",
		"Complex.matching.pattern",
		"Complex.anything.pattern",
		"Complex.prefixonesuffix.pattern",
		"Complex.prefixtwofix.pattern",
		"Complex.anything.pattern",
		"Question.1at_begin",
		"Question.at_the_end2",
	}

	metrics2 := metrics.ConfigureFilterMetrics("test")

	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Scheduler")

	Convey("Create new pattern storage, GetPatterns returns error, should error", t, func() {
		database.EXPECT().GetPatterns().Return(nil, fmt.Errorf("Some error here"))
		_, err := NewPatternStorage(database, metrics2, logger)
		So(err, ShouldBeError, fmt.Errorf("Some error here"))
	})

	database.EXPECT().GetPatterns().Return(testPatterns, nil)
	patternsStorage, err := NewPatternStorage(database, metrics2, logger)

	Convey("Create new pattern storage, should no error", t, func() {
		So(err, ShouldBeEmpty)
	})

	Convey("When invalid metric arrives, should be properly counted", t, func() {
		matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("Invalid.metric"))
		So(matchedMetrics, ShouldBeNil)
		So(metrics2.TotalMetricsReceived.Count(), ShouldEqual, 1)
		So(metrics2.ValidMetricsReceived.Count(), ShouldEqual, 0)
		So(metrics2.MatchingMetricsReceived.Count(), ShouldEqual, 0)
	})

	Convey("When valid non-matching metric arrives", t, func() {
		patternsStorage.metrics = metrics.ConfigureFilterMetrics("test")
		Convey("When metric arrives with int64 timestamp", func() {
			for _, metric := range nonMatchingMetrics {
				matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte(metric + " 12 1234567890"))
				So(matchedMetrics, ShouldBeNil)
			}
			So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, len(nonMatchingMetrics))
			So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, len(nonMatchingMetrics))
			So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, 0)
		})

		Convey("When metric arrives with float64 timestamp", func() {
			patternsStorage.metrics = metrics.ConfigureFilterMetrics("test")
			for _, metric := range nonMatchingMetrics {
				matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte(metric + " 12 1234567890.0"))
				So(matchedMetrics, ShouldBeNil)
			}
			So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, len(nonMatchingMetrics))
			So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, len(nonMatchingMetrics))
			So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, 0)
		})
	})

	Convey("When valid matching metric arrives", t, func() {
		patternsStorage.metrics = metrics.ConfigureFilterMetrics("test")
		Convey("When metric name is pure", func() {
			for _, metric := range matchingMetrics {
				matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte(metric + " 12 1234567890"))
				So(matchedMetrics, ShouldNotBeNil)
			}
			So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, len(matchingMetrics))
			So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, len(matchingMetrics))
			So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, len(matchingMetrics))
		})

		patternsStorage.metrics = metrics.ConfigureFilterMetrics("test")
		Convey("When value has dot", func() {
			for _, metric := range matchingMetrics {
				matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte(metric + " 12.000000 1234567890"))
				So(matchedMetrics, ShouldNotBeNil)
			}
			So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, len(matchingMetrics))
			So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, len(matchingMetrics))
			So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, len(matchingMetrics))
		})

		patternsStorage.metrics = metrics.ConfigureFilterMetrics("test")
		Convey("When timestamp is float64", func() {
			for _, metric := range matchingMetrics {
				matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte(metric + " 12 1234567890.0"))
				So(matchedMetrics, ShouldNotBeNil)
			}
			So(patternsStorage.metrics.TotalMetricsReceived.Count(), ShouldEqual, len(matchingMetrics))
			So(patternsStorage.metrics.ValidMetricsReceived.Count(), ShouldEqual, len(matchingMetrics))
			So(patternsStorage.metrics.MatchingMetricsReceived.Count(), ShouldEqual, len(matchingMetrics))
		})
	})

	mockCtrl.Finish()
}
