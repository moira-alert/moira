package plotting

import (
	"github.com/golang/freetype/truetype"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

// Plot represents plot structure to render
type Plot struct {
	Title      string
	Theme      string
	Rising    *bool
	WarnValue  *float64
	ErrorValue *float64
}

// FromParams returns Plot struct
func FromParams(plotTitle string, plotTheme string, isRising *bool, warnValue *float64, errorValue *float64) Plot {
	return Plot{plotTitle, plotTheme, isRising, warnValue, errorValue}
}

// IsRaising returns true if plot is of type Raising
func (plot Plot) IsRising() bool {
	if plot.Rising != nil {
		return *plot.Rising
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
	// TODO: Return "no metrics found" as picture too

	rising := plot.IsRising()
	yAxisMain, yAxisDescending := GetYAxisParams(rising)

	plotSeries := make([]chart.Series, 0)

	for timeSerieIndex := range metricsData {
		plotCurves := GeneratePlotCurves(metricsData[timeSerieIndex], timeSerieIndex, yAxisMain)
		for _, timeSerie := range plotCurves {
			plotSeries = append(plotSeries, timeSerie)
		}
	}

	plotLimits := ResolveLimits(metricsData)
	plotThresholds := GenerateThresholds(plot, plotLimits, rising)

	for _, plotThreshold := range plotThresholds {
		plotSeries = append(plotSeries, plotThreshold.GenerateThresholdSeries(plotLimits, rising))
		plotSeries = append(plotSeries, plotThreshold.GenerateAnnotationSeries(plotLimits, rising, plotFont))
	}

	bgPadding := GetBgPadding(plotLimits, len(plotThresholds))
	gridStyle := GetGridStyle(plot.Theme)

	yAxisValuesFormatter := GetYAxisValuesFormatter(plotLimits)

	renderable := chart.Chart{

		Title: SanitizeLabelName(plot.Title, 40),
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
