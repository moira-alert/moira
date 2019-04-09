package metricSource

import (
	"fmt"
	"math"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMakeMetricData(t *testing.T) {
	Convey("Just make MetricData", t, func(c C) {
		metricData := MakeMetricData("123", []float64{1, 2, 3}, 60, 0)
		c.So(*metricData, ShouldResemble, MetricData{
			Name:      "123",
			Values:    []float64{1, 2, 3},
			StepTime:  60,
			StartTime: 0,
			StopTime:  180,
		})

		metricData = MakeMetricData("123", []float64{1, 2, 3}, 10, 0)
		c.So(*metricData, ShouldResemble, MetricData{
			Name:      "123",
			Values:    []float64{1, 2, 3},
			StepTime:  10,
			StartTime: 0,
			StopTime:  30,
		})

		metricData = MakeMetricData("000", make([]float64, 0), 10, 0)
		c.So(*metricData, ShouldResemble, MetricData{
			Name:      "000",
			Values:    make([]float64, 0),
			StepTime:  10,
			StartTime: 0,
			StopTime:  0,
		})
	})

	Convey("Make empty metric data", t, func(c C) {
		metricData := MakeEmptyMetricData("123", 10, 50, 100)
		c.So(metricData.Values, ShouldHaveLength, 5)

		metricData = MakeEmptyMetricData("123", 10, 51, 99)
		c.So(metricData.Values, ShouldHaveLength, 5)

		metricData = MakeEmptyMetricData("123", 10, 51, 102)
		c.So(metricData.Values, ShouldHaveLength, 6)

		metricData = MakeEmptyMetricData("123", 60, 51, 102)
		c.So(metricData.Values, ShouldHaveLength, 1)

		metricData = MakeEmptyMetricData("123", 40, 51, 102)
		c.So(metricData.Values, ShouldHaveLength, 2)
	})
}

func TestGetTimestampValue(t *testing.T) {
	Convey("IsAbsent only false", t, func(c C) {
		metricData := MetricData{
			Name:      "m",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{0, 1, 2, 3, 4},
		}
		Convey("Has value", t, func(c C) {
			actual := metricData.GetTimestampValue(18)
			c.So(actual, ShouldEqual, 0)
			actual = metricData.GetTimestampValue(17)
			c.So(actual, ShouldEqual, 0)
			actual = metricData.GetTimestampValue(24)
			c.So(actual, ShouldEqual, 0)
			actual = metricData.GetTimestampValue(36)
			c.So(actual, ShouldEqual, 1)
			actual = metricData.GetTimestampValue(37)
			c.So(actual, ShouldEqual, 2)
			actual = metricData.GetTimestampValue(66)
			c.So(actual, ShouldEqual, 4)
		})

		Convey("No value", t, func(c C) {
			actual := metricData.GetTimestampValue(16)
			c.So(math.IsNaN(actual), ShouldBeTrue)
			actual = metricData.GetTimestampValue(67)
			c.So(math.IsNaN(actual), ShouldBeTrue)
		})
	})

	Convey("Values has nodata points", t, func(c C) {
		metricData := MetricData{
			Name:      "m",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{0, math.NaN(), math.NaN(), 3, math.NaN()},
		}

		actual := metricData.GetTimestampValue(18)
		c.So(actual, ShouldEqual, 0)
		actual = metricData.GetTimestampValue(27)
		c.So(math.IsNaN(actual), ShouldBeTrue)
		actual = metricData.GetTimestampValue(30)
		c.So(math.IsNaN(actual), ShouldBeTrue)
		actual = metricData.GetTimestampValue(39)
		c.So(math.IsNaN(actual), ShouldBeTrue)
		actual = metricData.GetTimestampValue(49)
		c.So(actual, ShouldEqual, 3)
		actual = metricData.GetTimestampValue(57)
		c.So(math.IsNaN(actual), ShouldBeTrue)
		actual = metricData.GetTimestampValue(66)
		c.So(math.IsNaN(actual), ShouldBeTrue)
	})
}

func TestMetricData_String(t *testing.T) {
	metricData1 := MakeMetricData("123", []float64{1, 2, 3}, 60, 0)
	metricData2 := MakeEmptyMetricData("123", 10, 50, 100)
	Convey("MetricData with points", t, func(c C) {
		c.So(metricData1.String(), ShouldResemble, "Metric: 123, StartTime: 0, StopTime: 180, StepTime: 60, Points: [1 2 3]")
	})

	Convey("MetricData with NaN points", t, func(c C) {
		c.So(metricData2.String(), ShouldResemble, "Metric: 123, StartTime: 50, StopTime: 100, StepTime: 10, Points: [NaN NaN NaN NaN NaN]")
	})

	Convey("MetricsData array", t, func(c C) {
		c.So(fmt.Sprintf("%v", []*MetricData{metricData1, metricData2}), ShouldResemble, "[Metric: 123, StartTime: 0, StopTime: 180, StepTime: 60, Points: [1 2 3] Metric: 123, StartTime: 50, StopTime: 100, StepTime: 10, Points: [NaN NaN NaN NaN NaN]]")
	})
}
