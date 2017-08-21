package target

import (
	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
	. "github.com/smartystreets/goconvey/convey"
	"math"
	"testing"
)

func TestGetTimestampValue(t *testing.T) {
	Convey("IsAbsent only false", t, func() {
		fetchResponse := pb.FetchResponse{
			Name:      "m",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{0, 1, 2, 3, 4},
			IsAbsent:  []bool{false, false, false, false, false},
		}
		timeSeries := TimeSeries{FetchResponse: fetchResponse}
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

	Convey("IsAbsent has true", t, func() {
		fetchResponse := pb.FetchResponse{
			Name:      "m",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{0, 1, 2, 3, 4},
			IsAbsent:  []bool{false, true, true, false, true},
		}
		timeSeries := TimeSeries{FetchResponse: fetchResponse}

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
