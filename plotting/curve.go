package plotting

import (
	"time"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

// PlotCurve is a single curve for given timeserie
type PlotCurve struct {
	TimeStamps []time.Time
	Values     []float64
}

// GeneratePlotCurves returns go-chart timeseries to generate plot curves
func GeneratePlotCurves(metricData *types.MetricData, curveColor int, mainYAxis int) []chart.TimeSeries {
	curves := DescribePlotCurves(metricData)
	curveSeries := make([]chart.TimeSeries, 0)
	if curveColor > len(CurveColors) {
		curveColor = 1
	}
	for _, curve := range curves {
		if len(curve.Values) > 1 {
			curveSerie := chart.TimeSeries{
				Name: metricData.Name,
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
	return curveSeries
}

// DescribePlotCurves returns parameters for required curves
func DescribePlotCurves(metricData *types.MetricData) []PlotCurve {
	curves := []PlotCurve{{}}
	curvesInd := 0

	values := make(chan float64, len(metricData.Values))
	for _, val := range metricData.Values {
		values <- val
	}

	start, timeStamp := ResolveFirstPoint(metricData)

	for absInd := start; absInd < len(metricData.IsAbsent); absInd++ {
		switch metricData.IsAbsent[absInd] {
		case false:
			curves[curvesInd].TimeStamps = append(curves[curvesInd].TimeStamps, Int32ToTime(timeStamp))
			curves[curvesInd].Values = append(curves[curvesInd].Values, <-values)
		case true:
			if len(curves[curvesInd].Values) > 0 {
				curves = append(curves, PlotCurve{})
				curvesInd += 1
			}
		}
		timeStamp += metricData.StepTime
	}
	return curves
}

// ResolveFirstPoint returns first point coordinates
func ResolveFirstPoint(metricData *types.MetricData) (int, int32) {
	start := 0
	startTime := metricData.StartTime
	for _, absVal := range metricData.IsAbsent {
		if absVal == true {
			start += 1
			startTime += metricData.StepTime
		} else {
			break
		}
	}
	return start, startTime
}
