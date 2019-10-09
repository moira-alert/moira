package plotting

import (
	"math"
	"time"

	"github.com/beevee/go-chart"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
)

// plotCurve is a single curve for given timeserie
type plotCurve struct {
	timeStamps []time.Time
	values     []float64
}

// getCurveSeriesList returns curve series list
func getCurveSeriesList(metricsData []metricSource.MetricData, theme moira.PlotTheme) []chart.TimeSeries {
	curveSeriesList := make([]chart.TimeSeries, 0)
	for metricDataInd := range metricsData {
		curveStyle, pointStyle := theme.GetSerieStyles(metricDataInd)
		curveSeries := generatePlotCurves(metricsData[metricDataInd], curveStyle, pointStyle)
		curveSeriesList = append(curveSeriesList, curveSeries...)
	}
	return curveSeriesList
}

// generatePlotCurves returns go-chart timeseries to generate plot curves
func generatePlotCurves(metricData metricSource.MetricData, curveStyle chart.Style, pointStyle chart.Style) []chart.TimeSeries {
	curves := describePlotCurves(metricData)
	curveSeries := make([]chart.TimeSeries, 0)
	for _, curve := range curves {
		var serieStyle chart.Style
		switch len(curve.values) {
		case 0:
			continue
		case 1:
			serieStyle = pointStyle
		default:
			serieStyle = curveStyle
		}
		curveSerie := chart.TimeSeries{
			Name:    metricData.Name,
			YAxis:   chart.YAxisSecondary,
			Style:   serieStyle,
			XValues: curve.timeStamps,
			YValues: curve.values,
		}
		curveSeries = append(curveSeries, curveSerie)
	}
	return curveSeries
}

// describePlotCurves returns parameters for required curves
func describePlotCurves(metricData metricSource.MetricData) []plotCurve {
	curves := []plotCurve{{}}
	curvesInd := 0

	start, timeStamp := resolveFirstPoint(metricData)

	for valInd := start; valInd < len(metricData.Values); valInd++ {
		pointValue := metricData.Values[valInd]
		if !math.IsNaN(pointValue) {
			timeStampValue := moira.Int64ToTime(timeStamp)
			curves[curvesInd].timeStamps = append(curves[curvesInd].timeStamps, timeStampValue)
			curves[curvesInd].values = append(curves[curvesInd].values, pointValue)
		} else {
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
func resolveFirstPoint(metricData metricSource.MetricData) (int, int64) {
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
