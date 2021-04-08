package conversion

import (
	"strings"
	"testing"

	metricsource "github.com/moira-alert/moira/metric_source"
	. "github.com/smartystreets/goconvey/convey"
)

func TestErrUnexpectedAloneMetric_Error(t *testing.T) {
	Convey("ErrUnexpectedAloneMetric message", t, func() {
		declared := map[string]bool{
			"t1": true,
			"t2": true,
		}
		unexpected := map[string][]string{
			"t2": {"metric.test.1", "metric.test.unexpected"},
		}

		want := strings.ReplaceAll(`Unexpected to have some targets with more than only one metric.
		Expected targets with only one metric: t1, t2
		Targets with multiple metrics but that declared as targets with alone metrics:
			t2 — metric.test.1, metric.test.unexpected`, "\n\t\t", "\n")

		err := ErrUnexpectedAloneMetric{
			declared:   declared,
			unexpected: unexpected,
		}
		So(err.Error(), ShouldResemble, want)
	})
}

func TestErrUnexpectedAloneMetricBuilder(t *testing.T) {
	Convey("errUnexpectedAloneMetricBuilder", t, func() {
		builder := newErrUnexpectedAloneMetricBuilder()

		Convey("constructor", func() {
			So(builder, ShouldResemble, errUnexpectedAloneMetricBuilder{
				returnError: false,
				result:      ErrUnexpectedAloneMetric{},
			})
		})
		Convey("set declared", func() {
			builder.setDeclared(map[string]bool{"t1": true})
			So(builder, ShouldResemble, errUnexpectedAloneMetricBuilder{
				returnError: false,
				result: ErrUnexpectedAloneMetric{
					declared: map[string]bool{"t1": true},
				},
			})
		})
		Convey("add unexpected", func() {
			builder.setDeclared(map[string]bool{"t1": true})
			builder.addUnexpected("t1", map[string]metricsource.MetricData{"metric.test.1": {Name: "metric.test.1"}, "metric.test.unexpected": {Name: "metric.test.unexpected"}})
			So(builder.returnError, ShouldBeTrue)
			So(builder.result.declared, ShouldResemble, map[string]bool{"t1": true})
			So(builder.result.unexpected, ShouldContainKey, "t1")
			So(builder.result.unexpected["t1"], ShouldHaveLength, 2)
			So(builder.result.unexpected["t1"], ShouldContain, "metric.test.1")
			So(builder.result.unexpected["t1"], ShouldContain, "metric.test.unexpected")
		})
		Convey("build", func() {
			Convey("error returned", func() {
				builder.setDeclared(map[string]bool{"t1": true})
				builder.addUnexpected("t1", map[string]metricsource.MetricData{"metric.test.1": {Name: "metric.test.1"}, "metric.test.unexpected": {Name: "metric.test.unexpected"}})
				err := builder.build()
				So(err.(ErrUnexpectedAloneMetric).declared, ShouldResemble, map[string]bool{"t1": true})
				So(err.(ErrUnexpectedAloneMetric).unexpected, ShouldContainKey, "t1")
				So(err.(ErrUnexpectedAloneMetric).unexpected["t1"], ShouldHaveLength, 2)
				So(err.(ErrUnexpectedAloneMetric).unexpected["t1"], ShouldContain, "metric.test.1")
				So(err.(ErrUnexpectedAloneMetric).unexpected["t1"], ShouldContain, "metric.test.unexpected")
			})
			Convey("nil returned", func() {
				err := builder.build()
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestErrEmptyAloneMetricsTarget_Error(t *testing.T) {
	Convey("ErrEmptyAloneMetricsTarget", t, func() {
		err := NewErrEmptyAloneMetricsTarget("t1")
		So(err.Error(), ShouldResemble, "target t1 declared as alone metrics target but do not have any metrics and saved state in last check")
	})
}
