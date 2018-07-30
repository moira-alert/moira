package target

import (
	"math"
	"testing"

	"github.com/go-graphite/carbonapi/expr/types"
	pb "github.com/go-graphite/protocol/carbonapi_v3_pb"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetTimestampValue(t *testing.T) {
	Convey("IsAbsent only false", t, func() {
		fetchResponse := pb.FetchResponse{
			Name:      "m",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{0, 1, 2, 3, 4},
		}
		timeSeries := TimeSeries{
			MetricData: types.MetricData{FetchResponse: fetchResponse},
		}
		Convey("Has value", func() {
			actual := timeSeries.GetTimestampValue(18)
			So(actual, ShouldEqual, 0)
			actual = timeSeries.GetTimestampValue(17)
			So(actual, ShouldEqual, 0)
			actual = timeSeries.GetTimestampValue(24)
			So(actual, ShouldEqual, 0)
			actual = timeSeries.GetTimestampValue(36)
			So(actual, ShouldEqual, 1)
			actual = timeSeries.GetTimestampValue(37)
			So(actual, ShouldEqual, 2)
			actual = timeSeries.GetTimestampValue(66)
			So(actual, ShouldEqual, 4)
		})

		Convey("No value", func() {
			actual := timeSeries.GetTimestampValue(16)
			So(math.IsNaN(actual), ShouldBeTrue)
			actual = timeSeries.GetTimestampValue(67)
			So(math.IsNaN(actual), ShouldBeTrue)
		})
	})

	Convey("Values has nodata points", t, func() {
		fetchResponse := pb.FetchResponse{
			Name:      "m",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{0, math.NaN(), math.NaN(), 3, math.NaN()},
		}

		timeSeries := TimeSeries{
			MetricData: types.MetricData{FetchResponse: fetchResponse},
		}

		actual := timeSeries.GetTimestampValue(18)
		So(actual, ShouldEqual, 0)
		actual = timeSeries.GetTimestampValue(27)
		So(math.IsNaN(actual), ShouldBeTrue)
		actual = timeSeries.GetTimestampValue(30)
		So(math.IsNaN(actual), ShouldBeTrue)
		actual = timeSeries.GetTimestampValue(39)
		So(math.IsNaN(actual), ShouldBeTrue)
		actual = timeSeries.GetTimestampValue(49)
		So(actual, ShouldEqual, 3)
		actual = timeSeries.GetTimestampValue(57)
		So(math.IsNaN(actual), ShouldBeTrue)
		actual = timeSeries.GetTimestampValue(66)
		So(math.IsNaN(actual), ShouldBeTrue)
	})
}
