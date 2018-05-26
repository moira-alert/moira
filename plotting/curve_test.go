package plotting

import (
	"math"
	"testing"
	"time"

	"github.com/go-graphite/carbonapi/expr/types"
	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	fetchResponse = pb.FetchResponse{
		StartTime: int32(0),
		StepTime:  int32(10),
		StopTime:  int32(100),
	}
	firstValIsAbsentVals = []float64{
		math.NaN(), math.NaN(), 32, math.NaN(), 54, 20, 43, 56, 2, 79, 76,
	}
	firstValIsPresentVals = []float64{
		11, 23, 45, math.NaN(), 47, math.NaN(), 32, 65, 78, 76, 74,
	}
)

// TestResolveFirstPoint tests ResolveFirstPoint returns correct start time
// for given MetricData whether IsAbsent[0] is true or false
func TestResolveFirstPoint(t *testing.T) {
	metricData := types.MetricData{FetchResponse: fetchResponse}
	Convey("First value is absent", t, func() {
		metricData.Values = firstValIsAbsentVals
		firstPointInd, startTime := ResolveFirstPoint(&metricData)
		So(firstPointInd, ShouldEqual, 2)
		So(startTime, ShouldEqual, 20)
	})
	Convey("First value is present", t, func() {
		metricData.Values = firstValIsPresentVals
		firstPointInd, startTime := ResolveFirstPoint(&metricData)
		So(firstPointInd, ShouldEqual, 0)
		So(startTime, ShouldEqual, 0)
	})
}

// TestDescribePlotCurves tests DescribePlotCurves returns collection of
// n PlotCurves from timeseries with n-1 gaps (IsAbsent values)
func TestDescribePlotCurves(t *testing.T) {
	metricData := types.MetricData{FetchResponse: fetchResponse}
	Convey("First value is absent", t, func() {
		metricData.Values = firstValIsAbsentVals
		plotCurves := DescribePlotCurves(&metricData)
		So(len(plotCurves), ShouldEqual, 2)
		So(plotCurves[0].Values, ShouldResemble, []float64{32})
		So(plotCurves[0].TimeStamps, ShouldResemble, []time.Time{
			Int32ToTime(20),
		})
		So(plotCurves[1].Values, ShouldResemble, []float64{54, 20, 43, 56, 2, 79, 76})
		So(plotCurves[1].TimeStamps, ShouldResemble, []time.Time{
			Int32ToTime(40),
			Int32ToTime(50),
			Int32ToTime(60),
			Int32ToTime(70),
			Int32ToTime(80),
			Int32ToTime(90),
			Int32ToTime(100),
		})
	})
	Convey("First value is present", t, func() {
		metricData.Values = firstValIsPresentVals
		plotCurves := DescribePlotCurves(&metricData)
		So(len(plotCurves), ShouldEqual, 3)
		So(plotCurves[0].Values, ShouldResemble, []float64{11, 23, 45})
		So(plotCurves[0].TimeStamps, ShouldResemble, []time.Time{
			Int32ToTime(0),
			Int32ToTime(10),
			Int32ToTime(20),
		})
		So(plotCurves[1].Values, ShouldResemble, []float64{47})
		So(plotCurves[1].TimeStamps, ShouldResemble, []time.Time{
			Int32ToTime(40),
		})
		So(plotCurves[2].Values, ShouldResemble, []float64{32, 65, 78, 76, 74})
		So(plotCurves[2].TimeStamps, ShouldResemble, []time.Time{
			Int32ToTime(60),
			Int32ToTime(70),
			Int32ToTime(80),
			Int32ToTime(90),
			Int32ToTime(100),
		})
	})
}

// TestGeneratePlotCurves tests GeneratePlotCurves returns
// collection of chart.Timeseries with actual field values
func TestGeneratePlotCurves(t *testing.T) {
	metricData := types.MetricData{FetchResponse: fetchResponse}
	Convey("First value is absent", t, func() {
		metricName := "metric.firstValueIsAbsent"
		metricData.FetchResponse.Name = metricName
		metricData.Values = firstValIsAbsentVals
		curveSeries := GeneratePlotCurves(&metricData, 0, 0)
		So(curveSeries[0].Name, ShouldEqual, metricName)
		So(curveSeries[0].YValues, ShouldResemble, []float64{54, 20, 43, 56, 2, 79, 76})
		So(curveSeries[0].XValues, ShouldResemble, []time.Time{
			Int32ToTime(40),
			Int32ToTime(50),
			Int32ToTime(60),
			Int32ToTime(70),
			Int32ToTime(80),
			Int32ToTime(90),
			Int32ToTime(100),
		})
	})
	Convey("First value is present", t, func() {
		metricName := "metric.firstValueIsPresent"
		metricData.FetchResponse.Name = metricName
		metricData.Values = firstValIsPresentVals
		curveSeries := GeneratePlotCurves(&metricData, 0, 0)
		So(len(curveSeries), ShouldEqual, 2)
		So(curveSeries[0].Name, ShouldEqual, metricName)
		So(curveSeries[0].YValues, ShouldResemble, []float64{11, 23, 45})
		So(curveSeries[0].XValues, ShouldResemble, []time.Time{
			Int32ToTime(0),
			Int32ToTime(10),
			Int32ToTime(20),
		})
		So(curveSeries[1].Name, ShouldEqual, metricName)
		So(curveSeries[1].YValues, ShouldResemble, []float64{32, 65, 78, 76, 74})
		So(curveSeries[1].XValues, ShouldResemble, []time.Time{
			Int32ToTime(60),
			Int32ToTime(70),
			Int32ToTime(80),
			Int32ToTime(90),
			Int32ToTime(100),
		})
	})
}
