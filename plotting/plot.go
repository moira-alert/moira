package plotting

import (
	"github.com/golang/freetype/truetype"
	"time"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
	"github.com/wcharczuk/go-chart/util"
)

// Plot represents plot structure to render
type Plot struct {
	Title      string
	Theme      string
	Raising    *bool
	WarnValue  *float64
	ErrorValue *float64
}

// FromParams returns Plot struct
func FromParams(plotTitle string, plotTheme string, isRaising *bool, warnValue *float64, errorValue *float64) Plot {
	return Plot{plotTitle, plotTheme, isRaising, warnValue, errorValue}
}

// IsRaising returns true if plot is of type Raising
func (plot Plot) IsRaising() bool {
	if plot.Raising != nil {
		return *plot.Raising
	}
	if plot.ErrorValue != nil && plot.WarnValue != nil {
		if *plot.ErrorValue > *plot.WarnValue {
			return true
		}
	}
	return false
}

// GetRenderable returns go-chart to render
func (plot Plot) GetRenderable(metricsData []*types.MetricData, plotFont *truetype.Font) chart.Chart {

	raising := plot.IsRaising()
	yAxisMain, yAxisDescending := GetYAxisParams(raising)

	plotSeries := make([]chart.Series, 0)

	// TODO: use isolated method to securely count plot limits

	var plotFrom time.Time
	var plotTo time.Time
	var plotLowest float64
	var plotHighest float64

	for timeSerieIndex := range metricsData {
		plotCurves, curveTimeLimits, curvesValueLimits := GeneratePlotCurves(metricsData[timeSerieIndex], timeSerieIndex, yAxisMain)
		for _, timeLimit := range curveTimeLimits {
			plotFrom, plotTo = util.Math.MinAndMaxOfTime(plotFrom, plotTo, timeLimit)
		}
		for _, valueLimit := range curvesValueLimits {
			plotLowest, plotHighest = util.Math.MinAndMax(plotLowest, plotHighest, valueLimit)
		}
		for _, timeSerie := range plotCurves {
			plotSeries = append(plotSeries, timeSerie)
		}
	}

	plotLimits := Limits{
		From:    plotFrom,
		To:      plotTo,
		Lowest:  plotLowest,
		Highest: plotHighest,
	}

	plotThresholds := GenerateThresholds(plot, plotLimits)

	for _, plotThreshold := range plotThresholds {
		plotSeries = append(plotSeries, plotThreshold.GenerateThresholdSeries(plotLimits, raising))
		plotSeries = append(plotSeries, plotThreshold.GenerateAnnotationSeries(plotLimits, raising, plotFont))
	}

	bgPadding := GetBgPadding(len(plotThresholds))
	gridStyle := GetGridStyle(plot.Theme)

	yAxisValuesFormatter := GetYAxisValuesFormatter(plotLimits)

	renderable := chart.Chart{

		Title: plot.Title,
		TitleStyle: chart.Style{
			Show:        true,
			Font:        plotFont,
			FontColor:   chart.ColorAlternateGray,
			FillColor:   drawing.ColorFromHex(plot.Theme),
			StrokeColor: drawing.ColorFromHex(plot.Theme),
		},

		Width:  PlotWidth,
		Height: PlotHeight,

		Canvas: chart.Style{
			FillColor: drawing.ColorFromHex(plot.Theme),
		},
		Background: chart.Style{
			FillColor: drawing.ColorFromHex(plot.Theme),
			Padding:   bgPadding,
		},

		XAxis: chart.XAxis{
			Style: chart.Style{
				Show:        true,
				Font:        plotFont,
				FontSize:    8,
				FontColor:   chart.ColorAlternateGray,
				StrokeColor: drawing.ColorFromHex(plot.Theme),
			},
			GridMinorStyle: gridStyle,
			GridMajorStyle: gridStyle,

			ValueFormatter: chart.TimeValueFormatterWithFormat("15:04"),
		},

		YAxis: chart.YAxis{
			Style: chart.Style{
				Show: false,
			},
			GridMinorStyle: gridStyle,
			GridMajorStyle: gridStyle,

			Range: &chart.ContinuousRange{
				Descending: yAxisDescending,
				Max:        plotLimits.Highest,
				Min:        0,
			},
		},

		YAxisSecondary: chart.YAxis{
			ValueFormatter: yAxisValuesFormatter,
			Style: chart.Style{
				Show:        true,
				Font:        plotFont,
				FontColor:   chart.ColorAlternateGray,
				StrokeColor: drawing.ColorFromHex(plot.Theme),
			},
			GridMinorStyle: gridStyle,
			GridMajorStyle: gridStyle,

			Range: &chart.ContinuousRange{
				Max: plotLimits.Highest,
				Min: plotLimits.Lowest,
			},
		},

		Series: plotSeries,
	}

	renderable.Elements = []chart.Renderable{
		GetPlotLegend(&renderable),
	}

	return renderable
}
