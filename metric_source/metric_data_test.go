package metricSource

import (
	"math"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetTimestampValue(t *testing.T) {
	Convey("IsAbsent only false", t, func() {
		metricData := MetricData{
			Name:      "m",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{0, 1, 2, 3, 4},
		}
		Convey("Has value", func() {
			actual := metricData.GetTimestampValue(18)
			So(actual, ShouldEqual, 0)
			actual = metricData.GetTimestampValue(17)
			So(actual, ShouldEqual, 0)
			actual = metricData.GetTimestampValue(24)
			So(actual, ShouldEqual, 0)
			actual = metricData.GetTimestampValue(36)
			So(actual, ShouldEqual, 1)
			actual = metricData.GetTimestampValue(37)
			So(actual, ShouldEqual, 2)
			actual = metricData.GetTimestampValue(66)
			So(actual, ShouldEqual, 4)
		})

		Convey("No value", func() {
			actual := metricData.GetTimestampValue(16)
			So(math.IsNaN(actual), ShouldBeTrue)
			actual = metricData.GetTimestampValue(67)
			So(math.IsNaN(actual), ShouldBeTrue)
		})
	})

	Convey("Values has nodata points", t, func() {
		metricData := MetricData{
			Name:      "m",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{0, math.NaN(), math.NaN(), 3, math.NaN()},
		}

		actual := metricData.GetTimestampValue(18)
		So(actual, ShouldEqual, 0)
		actual = metricData.GetTimestampValue(27)
		So(math.IsNaN(actual), ShouldBeTrue)
		actual = metricData.GetTimestampValue(30)
		So(math.IsNaN(actual), ShouldBeTrue)
		actual = metricData.GetTimestampValue(39)
		So(math.IsNaN(actual), ShouldBeTrue)
		actual = metricData.GetTimestampValue(49)
		So(actual, ShouldEqual, 3)
		actual = metricData.GetTimestampValue(57)
		So(math.IsNaN(actual), ShouldBeTrue)
		actual = metricData.GetTimestampValue(66)
		So(math.IsNaN(actual), ShouldBeTrue)
	})
}
