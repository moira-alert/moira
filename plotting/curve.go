package plotting

import (
	"math"
	"time"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
	"github.com/wcharczuk/go-chart/util"
)

// PlotCurve is a single curve for given timeserie
type PlotCurve struct {
	TimeStamps []time.Time
	Values     []float64
}

// GeneratePlotCurves returns go-chart timeseries to generate plot curves
func GeneratePlotCurves(metricData *types.MetricData, curveColor int, mainYAxis int) ([]chart.TimeSeries, []time.Time, []float64) {
	// TODO: create style to draw single value in between of gaps
	curves, timeLimits, valueLimits := DescribePlotCurves(metricData)
	curveSeries := make([]chart.TimeSeries, 0)
	if curveColor > len(CurveColors) {
		curveColor = 1
	}
	for _, curve := range curves {
		if len(curve.Values) > 1 {
			curveSerie := chart.TimeSeries{
				Name:  metricData.Name,
				YAxis: chart.YAxisType(mainYAxis),
				Style: chart.Style{
					Show:        true,
					StrokeWidth: 1,
					StrokeColor: drawing.ColorFromHex(CurveColors[curveColor]).WithAlpha(90),
					FillColor:   drawing.ColorFromHex(CurveColors[curveColor]).WithAlpha(20),
				},
				XValues: curve.TimeStamps,
				YValues: curve.Values,
			}
			curveSeries = append(curveSeries, curveSerie)
		}
	}
	return curveSeries, timeLimits, valueLimits
}

// DescribePlotCurves returns parameters for required curves
func DescribePlotCurves(metricData *types.MetricData) ([]PlotCurve, []time.Time, []float64) {
	curves := []PlotCurve{{}}
	curvesInd := 0

	values := make(chan float64, len(metricData.Values))
	for _, val := range metricData.Values {
		values <- val
	}

	start, timeStamp := ResolveFirstPoint(metricData)

	var pointValue float64
	var from time.Time
	var to time.Time
	var lowest float64
	var highest float64

	for absInd := start; absInd < len(metricData.IsAbsent); absInd++ {
		switch metricData.IsAbsent[absInd] {
		case false:
			pointValue = <-values
			if !math.IsNaN(pointValue) {
				timeStampValue := Int32ToTime(timeStamp)
				lowest, highest = util.Math.MinAndMax(lowest, highest, pointValue)
				from, to = util.Math.MinAndMaxOfTime(from, to, timeStampValue)
				curves[curvesInd].TimeStamps = append(curves[curvesInd].TimeStamps, timeStampValue)
				curves[curvesInd].Values = append(curves[curvesInd].Values, pointValue)
			}
		case true:
			if len(curves[curvesInd].Values) > 0 {
				curves = append(curves, PlotCurve{})
				curvesInd++
			}
		}
		timeStamp += metricData.StepTime
	}

	timeLimits := []time.Time{from, to}
	valueLimits := []float64{lowest, highest}

	return curves, timeLimits, valueLimits
}

// ResolveFirstPoint returns first point coordinates
func ResolveFirstPoint(metricData *types.MetricData) (int, int32) {
	start := 0
	startTime := metricData.StartTime
	for _, absVal := range metricData.IsAbsent {
		if absVal {
			start++
			startTime += metricData.StepTime
		} else {
			break
		}
	}
	return start, startTime
}
