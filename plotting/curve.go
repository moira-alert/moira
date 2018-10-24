package plotting

import (
	"math"
	"time"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

// plotCurve is a single curve for given timeserie
type plotCurve struct {
	timeStamps []time.Time
	values     []float64
}

func getCurveSeriesList(metricsData []*types.MetricData, theme *plotTheme, mainYAxis int, limitSeries []string) []chart.TimeSeries {
	curveSeriesList := make([]chart.TimeSeries, 0)
	for metricDataInd := range metricsData {
		if !mustBeShown(metricsData[metricDataInd].Name, limitSeries) {
			continue
		}
		curveColor := theme.pickCurveColor(metricDataInd)
		curveSeries := generatePlotCurves(metricsData[metricDataInd], curveColor, mainYAxis)
		for _, curveSerie := range curveSeries {
			curveSeriesList = append(curveSeriesList, curveSerie)
		}
	}
	return curveSeriesList
}

// generatePlotCurves returns go-chart timeseries to generate plot curves
func generatePlotCurves(metricData *types.MetricData, curveColor string, mainYAxis int) []chart.TimeSeries {
	// TODO: create style to draw single value in between of gaps
	curves := describePlotCurves(metricData)
	curveSeries := make([]chart.TimeSeries, 0)
	for _, curve := range curves {
		if len(curve.values) > 1 {
			curveSerie := chart.TimeSeries{
				Name:  metricData.Name,
				YAxis: chart.YAxisType(mainYAxis),
				Style: chart.Style{
					Show:        true,
					StrokeWidth: 1,
					StrokeColor: drawing.ColorFromHex(curveColor).WithAlpha(90),
					FillColor:   drawing.ColorFromHex(curveColor).WithAlpha(20),
				},
				XValues: curve.timeStamps,
				YValues: curve.values,
			}
			curveSeries = append(curveSeries, curveSerie)
		}
	}
	return curveSeries
}

// describePlotCurves returns parameters for required curves
func describePlotCurves(metricData *types.MetricData) []plotCurve {
	curves := []plotCurve{{}}
	curvesInd := 0

	start, timeStamp := resolveFirstPoint(metricData)

	for valInd := start; valInd < len(metricData.Values); valInd++ {
		pointValue := metricData.Values[valInd]
		switch math.IsNaN(pointValue) {
		case false:
			timeStampValue := int64ToTime(timeStamp)
			curves[curvesInd].timeStamps = append(curves[curvesInd].timeStamps, timeStampValue)
			curves[curvesInd].values = append(curves[curvesInd].values, pointValue)
		case true:
			if len(curves[curvesInd].values) > 0 {
				curves = append(curves, plotCurve{})
				curvesInd++
			}
		}
		timeStamp += metricData.StepTime
	}
	return curves
}

// resolveFirstPoint returns first point coordinates
func resolveFirstPoint(metricData *types.MetricData) (int, int64) {
	start := 0
	startTime := metricData.StartTime
	for _, metricVal := range metricData.Values {
		if math.IsNaN(metricVal) {
			start++
			startTime += metricData.StepTime
		} else {
			break
		}
	}
	return start, startTime
}

// mustBeShown returns true if metric must be shown
func mustBeShown(metricName string, metricsToShow []string) bool {
	if len(metricsToShow) == 0 {
		return true
	}
	for _, metricToShow := range metricsToShow {
		if metricToShow == metricName {
			return true
		}
	}
	return false
}
