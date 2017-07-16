package cache

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestParseMetricFromString(t *testing.T) {
	storage := PatternStorage{}

	Convey("Given invalid metric strings, should return errors", t, func() {
		invalidMetrics := []string{
			"Invalid.value 12g5 1234567890",
			"No.value.two.spaces  1234567890",
			"No.timestamp.space.in.the.end 12 ",
			"No.timestamp 12",
			" 12 1234567890",
			"Non-ascii.こんにちは 12 1234567890",
			"Non-printable.\000 12 1234567890",
			"",
			"\n",
			"Too.many.parts 1 2 3 4 12 1234567890",
			"Space.in.the.end 12 1234567890 ",
			" Space.in.the.beginning 12 1234567890",
			"\tNon-printable.in.the.beginning 12 1234567890",
			"\rNon-printable.in.the.beginning 12 1234567890",
			"Newline.in.the.end 12 1234567890\n",
			"Newline.in.the.end 12 1234567890\r",
			"Newline.in.the.end 12 1234567890\r\n",
		}

		for _, invalidMetric := range invalidMetrics {
			_, _, _, err := storage.parseMetricFromString([]byte(invalidMetric))
			So(err, ShouldBeError)
		}
	})

	Convey("Given valid metric strings, should return parsed values", t, func() {
		type ValidMetricCase struct {
			raw       string
			metric    string
			value     float64
			timestamp int64
		}
		validMetrics := []ValidMetricCase{
			{"One.two.three 123 1234567890", "One.two.three", 123, 1234567890},
			{"One.two.three 1.23e2 1234567890", "One.two.three", 123, 1234567890},
			{"One.two.three -123 1234567890", "One.two.three", -123, 1234567890},
			{"One.two.three +123 1234567890", "One.two.three", 123, 1234567890},
			{"One.two.three 123. 1234567890", "One.two.three", 123, 1234567890},
			{"One.two.three 123.0 1234567890", "One.two.three", 123, 1234567890},
			{"One.two.three .123 1234567890", "One.two.three", 0.123, 1234567890},
		}

		for _, validMetric := range validMetrics {
			metric, value, timestamp, err := storage.parseMetricFromString([]byte(validMetric.raw))
			So(err, ShouldBeEmpty)
			So(metric, ShouldResemble, []byte(validMetric.metric))
			So(value, ShouldResemble, validMetric.value)
			So(timestamp, ShouldResemble, validMetric.timestamp)
		}
	})
}

func TestProcessIncomingMetric(t *testing.T) {
	testPatterns := []string{
		"Simple.matching.pattern",
		"Star.single.*",
		"Star.*.double.any*",
		"Bracket.{one,two,three}.pattern",
		"Bracket.pr{one,two,three}suf",
		"Complex.matching.pattern",
		"Complex.*.*",
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

	metrics2 := metrics.ConfigureCacheMetrics()

	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Scheduler")

	Convey("Create new pattern storage, GetPatterns returns error, should error", t, func() {
		database.EXPECT().GetPatterns().Return(nil, fmt.Errorf("Some error here"))
		_, err := NewPatternStorage(database, metrics2, logger, true)
		So(err, ShouldBeError, fmt.Errorf("Some error here"))
	})

	database.EXPECT().GetPatterns().Return(testPatterns, nil)
	patternsStorage, err := NewPatternStorage(database, metrics2, logger, true)

	Convey("Create new pattern storage, should no error", t, func() {
		So(err, ShouldBeEmpty)
	})

	Convey("When invalid metric arrives, should be properly counted", t, func() {
		matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte("Invalid.metric"))
		So(matchedMetrics, ShouldBeNil)
		So(metrics2.TotalReceived, ShouldEqual, 1)
		So(metrics2.ValidReceived, ShouldEqual, 0)
		So(metrics2.MatchedReceived, ShouldEqual, 0)
	})

	patternsStorage.metrics = metrics.ConfigureCacheMetrics()

	Convey("When valid non-matching metric arrives", t, func() {
		Convey("When metric arrives with timestamp", func() {
			for _, metric := range nonMatchingMetrics {
				matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte(metric + " 12 1234567890"))
				So(matchedMetrics, ShouldBeNil)
			}
			So(patternsStorage.metrics.TotalReceived, ShouldEqual, len(nonMatchingMetrics))
			So(patternsStorage.metrics.ValidReceived, ShouldEqual, len(nonMatchingMetrics))
			So(patternsStorage.metrics.MatchedReceived, ShouldEqual, 0)
		})
	})

	Convey("When valid matching metric arrives", t, func() {
		patternsStorage.metrics = metrics.ConfigureCacheMetrics()
		Convey("When metric name is pure", func() {
			for _, metric := range matchingMetrics {
				matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte(metric + " 12 1234567890"))
				So(matchedMetrics, ShouldNotBeNil)
			}
			So(patternsStorage.metrics.TotalReceived, ShouldEqual, len(matchingMetrics))
			So(patternsStorage.metrics.ValidReceived, ShouldEqual, len(matchingMetrics))
			So(patternsStorage.metrics.MatchedReceived, ShouldEqual, len(matchingMetrics))
		})

		patternsStorage.metrics = metrics.ConfigureCacheMetrics()
		Convey("When value has dot", func() {
			for _, metric := range matchingMetrics {
				matchedMetrics := patternsStorage.ProcessIncomingMetric([]byte(metric + " 12.000000 1234567890"))
				So(matchedMetrics, ShouldNotBeNil)
			}
			So(patternsStorage.metrics.TotalReceived, ShouldEqual, len(matchingMetrics))
			So(patternsStorage.metrics.ValidReceived, ShouldEqual, len(matchingMetrics))
			So(patternsStorage.metrics.MatchedReceived, ShouldEqual, len(matchingMetrics))
		})
	})

	mockCtrl.Finish()
}
