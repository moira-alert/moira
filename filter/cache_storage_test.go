package filter

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
	"github.com/smartystreets/assertions/should"
	. "github.com/smartystreets/goconvey/convey"
	"strings"
	"testing"
)

var testRetentions = `
	# comment
	[simple]
	pattern = ^Simple\.
	retentions = 60s:2d,10m:30d,100m:90d

	[rare]
	pattern = suf$
	retentions = 20m:30d,8h:1y

	[hourly]
	pattern = hourly$
	retentions = 1h:1d

	[daily]
	pattern = daily$
	retentions = 1d:1w

	[weekly]
	pattern = weekly$
	retentions = 1w:1y

	[yearly]
	pattern = yearly$
	retentions = 1y:100y

	[default]
	pattern = .*
	retentions = 120:7d
	`

var expectedRetentionIntervals = []int{60, 1200, 3600, 86400, 604800, 31536000, 120}

var matchedMetrics = []moira.MatchedMetric{
	{
		Metric:             "Simple.matching.pattern",
		Patterns:           []string{"Simple.matching.pattern"},
		Value:              12,
		Timestamp:          31,
		RetentionTimestamp: 0,
		Retention:          60,
	},
	{
		Metric:             "Star.single.anything",
		Patterns:           []string{"Star.single.*"},
		Value:              12,
		Timestamp:          1234567890,
		RetentionTimestamp: 1234567890,
		Retention:          60,
	},
	{
		Metric:             "Star.anything.double.anything",
		Patterns:           []string{"Star.*.double.any*"},
		Value:              12,
		Timestamp:          1234567890,
		RetentionTimestamp: 1234567890,
		Retention:          60,
	},
	{
		Metric:             "Bracket.one.pattern",
		Patterns:           []string{"Bracket.{one,two,three}.pattern"},
		Value:              12,
		Timestamp:          1234567890,
		RetentionTimestamp: 1234567890,
		Retention:          60,
	},
	{
		Metric:             "Bracket.two.pattern",
		Patterns:           []string{"Bracket.{one,two,three}.pattern"},
		Value:              12,
		Timestamp:          1234567890,
		RetentionTimestamp: 1234567890,
		Retention:          60,
	},
	{
		Metric:             "Bracket.three.pattern",
		Patterns:           []string{"Bracket.{one,two,three}.pattern"},
		Value:              12,
		Timestamp:          1234567890,
		RetentionTimestamp: 1234567890,
		Retention:          60,
	},
	{
		Metric:             "Bracket.pronesuf",
		Patterns:           []string{"Bracket.pr{one,two,three}suf"},
		Value:              12,
		Timestamp:          600,
		RetentionTimestamp: 0,
		Retention:          60,
	},
	{
		Metric:             "Bracket.prtwosuf",
		Patterns:           []string{"Bracket.pr{one,two,three}suf"},
		Value:              12,
		Timestamp:          1234567890,
		RetentionTimestamp: 1234567890,
		Retention:          60,
	},
	{
		Metric:             "Bracket.prthreesuf",
		Patterns:           []string{"Bracket.pr{one,two,three}suf"},
		Value:              12,
		Timestamp:          1234567890,
		RetentionTimestamp: 1234567890,
		Retention:          60,
	},
	{
		Metric:             "Complex.matching.pattern",
		Patterns:           []string{"Complex.matching.pattern", "Complex.*.*"},
		Value:              12,
		Timestamp:          1234567890,
		RetentionTimestamp: 1234567890,
		Retention:          60,
	},
	{
		Metric:             "Complex.anything.pattern",
		Patterns:           []string{"Complex.*.*"},
		Value:              12,
		Timestamp:          1234567890,
		RetentionTimestamp: 1234567890,
		Retention:          60,
	},
	{
		Metric:             "Complex.prefixonesuffix.pattern",
		Patterns:           []string{"Complex.*.*", "Complex.*{one,two,three}suf*.pattern"},
		Value:              12,
		Timestamp:          1234567890,
		RetentionTimestamp: 1234567890,
		Retention:          60,
	},
	{
		Metric:             "Complex.prefixtwofix.pattern",
		Patterns:           []string{"Complex.*.*"},
		Value:              12,
		Timestamp:          1234567890,
		RetentionTimestamp: 1234567890,
		Retention:          60,
	},
	{
		Metric:             "Question.1at_begin",
		Patterns:           []string{"Question.?at_begin"},
		Value:              12,
		Timestamp:          1234567890,
		RetentionTimestamp: 1234567890,
		Retention:          60,
	},
	{
		Metric:             "Question.at_the_end2",
		Patterns:           []string{"Question.at_the_end?"},
		Value:              12,
		Timestamp:          151,
		RetentionTimestamp: 0,
		Retention:          60,
	},
}

func TestCacheStorage(t *testing.T) {
	metrics2 := metrics.ConfigureFilterMetrics("test")
	storage, err := NewCacheStorage(metrics2, strings.NewReader(testRetentions))

	Convey("Test good retentions", t, func() {
		So(err, ShouldBeEmpty)
		So(storage, ShouldNotBeNil)
		for i, retention := range storage.retentions {
			So(retention.retention, ShouldEqual, expectedRetentionIntervals[i])
		}
	})

	Convey("Test empty buffer and different metrics, should buffer len equal to matchedMetrics len", t, func() {
		buffer := make(map[string]*moira.MatchedMetric)
		for _, matchedMetric := range matchedMetrics {
			storage.EnrichMatchedMetric(buffer, &matchedMetric)
		}
		So(len(buffer), ShouldEqual, len(matchedMetrics))
	})

	storage, _ = NewCacheStorage(metrics2, strings.NewReader(testRetentions))

	Convey("Test add one metric twice, should buffer len is 1", t, func() {
		buffer := make(map[string]*moira.MatchedMetric)
		storage.EnrichMatchedMetric(buffer, &matchedMetrics[0])
		So(len(buffer), ShouldEqual, 1)
		storage.EnrichMatchedMetric(buffer, &matchedMetrics[0])
		So(len(buffer), ShouldEqual, 1)
	})
}

func TestRetentions(t *testing.T) {
	metrics2 := metrics.ConfigureFilterMetrics("test")
	storage, _ := NewCacheStorage(metrics2, strings.NewReader(testRetentions))

	Convey("Simple metric, should 60sec", t, func() {
		buffer := make(map[string]*moira.MatchedMetric)
		metr := matchedMetrics[0]

		storage.EnrichMatchedMetric(buffer, &metr)
		So(len(buffer), ShouldEqual, 1)
		So(metr.Retention, ShouldEqual, 60)
		So(metr.RetentionTimestamp, should.Equal, 60)
	})

	Convey("Suf metric, should 1200sec", t, func() {
		buffer := make(map[string]*moira.MatchedMetric)
		metr := matchedMetrics[6]

		storage.EnrichMatchedMetric(buffer, &metr)
		So(len(buffer), ShouldEqual, 1)
		So(metr.Retention, ShouldEqual, 1200)
		So(metr.RetentionTimestamp, should.Equal, 1200)
	})

	Convey("Default metric, should 120sec", t, func() {
		buffer := make(map[string]*moira.MatchedMetric)
		metr := matchedMetrics[14]

		storage.EnrichMatchedMetric(buffer, &metr)
		So(len(buffer), ShouldEqual, 1)
		So(metr.Retention, ShouldEqual, 120)
		So(metr.RetentionTimestamp, should.Equal, 120)
	})
}
