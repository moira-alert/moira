package plotting

import (
	"math"
	"testing"
	"time"

	"github.com/beevee/go-chart"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	metricData = metricSource.MetricData{
		StartTime: 0,
		StopTime:  100,
		StepTime:  10,
	}
	firstValIsAbsentVals = []float64{
		math.NaN(), math.NaN(), 32, math.NaN(), 54, 20, 43, 56, 2, 79, 76,
	}
	firstValIsPresentVals = []float64{
		11, 23, 45, math.NaN(), 47, math.NaN(), 32, 65, 78, 76, 74,
	}
)

// TestGeneratePlotCurves tests generatePlotCurves returns
// collection of chart.Timeseries with actual field values
func TestGeneratePlotCurves(t *testing.T) {
	Convey("First value is absent", t, func() {
		metricName := "metric.firstValueIsAbsent"
		metricData.Name = metricName
		metricData.Values = firstValIsAbsentVals
		curveSeries := generatePlotCurves(metricData, chart.Style{}, chart.Style{})
		So(len(curveSeries), ShouldEqual, 2)
		So(curveSeries[0].Name, ShouldEqual, metricName)
		So(curveSeries[0].YValues, ShouldResemble, []float64{32})
		So(curveSeries[0].XValues, ShouldResemble, []time.Time{
			moira.Int64ToTime(20),
		})
		So(curveSeries[1].Name, ShouldEqual, metricName)
		So(curveSeries[1].YValues, ShouldResemble, []float64{54, 20, 43, 56, 2, 79, 76})
		So(curveSeries[1].XValues, ShouldResemble, []time.Time{
			moira.Int64ToTime(40),
			moira.Int64ToTime(50),
			moira.Int64ToTime(60),
			moira.Int64ToTime(70),
			moira.Int64ToTime(80),
			moira.Int64ToTime(90),
			moira.Int64ToTime(100),
		})
	})
	Convey("First value is present", t, func() {
		metricName := "metric.firstValueIsPresent"
		metricData.Name = metricName
		metricData.Values = firstValIsPresentVals
		curveSeries := generatePlotCurves(metricData, chart.Style{}, chart.Style{})
		So(len(curveSeries), ShouldEqual, 3)
		So(curveSeries[0].Name, ShouldEqual, metricName)
		So(curveSeries[0].YValues, ShouldResemble, []float64{11, 23, 45})
		So(curveSeries[0].XValues, ShouldResemble, []time.Time{
			moira.Int64ToTime(0),
			moira.Int64ToTime(10),
			moira.Int64ToTime(20),
		})
		So(curveSeries[1].Name, ShouldEqual, metricName)
		So(curveSeries[1].YValues, ShouldResemble, []float64{47})
		So(curveSeries[1].XValues, ShouldResemble, []time.Time{
			moira.Int64ToTime(40),
		})
		So(curveSeries[2].Name, ShouldEqual, metricName)
		So(curveSeries[2].YValues, ShouldResemble, []float64{32, 65, 78, 76, 74})
		So(curveSeries[2].XValues, ShouldResemble, []time.Time{
			moira.Int64ToTime(60),
			moira.Int64ToTime(70),
			moira.Int64ToTime(80),
			moira.Int64ToTime(90),
			moira.Int64ToTime(100),
		})
	})
}

// TestDescribePlotCurves tests describePlotCurves returns collection of
// n PlotCurves from timeseries with n-1 gaps (IsAbsent values)
func TestDescribePlotCurves(t *testing.T) {
	Convey("First value is absent", t, func() {
		metricData.Values = firstValIsAbsentVals
		plotCurves := describePlotCurves(metricData)
		So(len(plotCurves), ShouldEqual, 2)
		So(plotCurves[0].values, ShouldResemble, []float64{32})
		So(plotCurves[0].timeStamps, ShouldResemble, []time.Time{
			moira.Int64ToTime(20),
		})
		So(plotCurves[1].values, ShouldResemble, []float64{54, 20, 43, 56, 2, 79, 76})
		So(plotCurves[1].timeStamps, ShouldResemble, []time.Time{
			moira.Int64ToTime(40),
			moira.Int64ToTime(50),
			moira.Int64ToTime(60),
			moira.Int64ToTime(70),
			moira.Int64ToTime(80),
			moira.Int64ToTime(90),
			moira.Int64ToTime(100),
		})
	})
	Convey("First value is present", t, func() {
		metricData.Values = firstValIsPresentVals
		plotCurves := describePlotCurves(metricData)
		So(len(plotCurves), ShouldEqual, 3)
		So(plotCurves[0].values, ShouldResemble, []float64{11, 23, 45})
		So(plotCurves[0].timeStamps, ShouldResemble, []time.Time{
			moira.Int64ToTime(0),
			moira.Int64ToTime(10),
			moira.Int64ToTime(20),
		})
		So(plotCurves[1].values, ShouldResemble, []float64{47})
		So(plotCurves[1].timeStamps, ShouldResemble, []time.Time{
			moira.Int64ToTime(40),
		})
		So(plotCurves[2].values, ShouldResemble, []float64{32, 65, 78, 76, 74})
		So(plotCurves[2].timeStamps, ShouldResemble, []time.Time{
			moira.Int64ToTime(60),
			moira.Int64ToTime(70),
			moira.Int64ToTime(80),
			moira.Int64ToTime(90),
			moira.Int64ToTime(100),
		})
	})
}

// TestResolveFirstPoint tests resolveFirstPoint returns correct start time
// for given MetricData whether IsAbsent[0] is true or false
func TestResolveFirstPoint(t *testing.T) {
	Convey("First value is absent", t, func() {
		metricData.Values = firstValIsAbsentVals
		firstPointInd, startTime := resolveFirstPoint(metricData)
		So(firstPointInd, ShouldEqual, 2)
		So(startTime, ShouldEqual, 20)
	})
	Convey("First value is present", t, func() {
		metricData.Values = firstValIsPresentVals
		firstPointInd, startTime := resolveFirstPoint(metricData)
		So(firstPointInd, ShouldEqual, 0)
		So(startTime, ShouldEqual, 0)
	})
}
