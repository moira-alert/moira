package plotting

import (
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/moira-alert/moira"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

// plot represents plot structure to render
type Plot struct {
	theme  *plotTheme
	width  int
	height int
}

// GetPlotTemplate returns plot template
func GetPlotTemplate(theme string) (*Plot, error) {
	plotTheme, err := getPlotTheme(theme)
	if err != nil {
		return nil, err
	}
	return &Plot{
		theme:  plotTheme,
		width:  800,
		height: 400,
	}, nil
}

// GetRenderable returns go-chart to render
func (plot *Plot) GetRenderable(trigger *moira.Trigger, metricsData []*types.MetricData, limitSeries []string) chart.Chart {
	// TODO: Return "no metrics found" as picture too

	yAxisMain, yAxisDescending := getYAxisParams(trigger.TriggerType)

	plotSeries := make([]chart.Series, 0)
	plotLimits := resolveLimits(metricsData)

	curveSeriesList := getCurveSeriesList(metricsData, plot.theme, yAxisMain, limitSeries)
	for _, curveSeries := range curveSeriesList {
		plotSeries = append(plotSeries, curveSeries)
	}

	thresholdSeriesList, hasThresholds := getThresholdSeriesList(trigger, plotLimits, plot.theme)
	for _, thresholdSeries := range thresholdSeriesList {
		plotSeries = append(plotSeries, thresholdSeries)
	}

	bgPadding := getBgPadding(plotLimits, hasThresholds)
	gridStyle := plot.theme.gridStyle

	yAxisValuesFormatter := getYAxisValuesFormatter(plotLimits)

	renderable := chart.Chart{

		Title: sanitizeLabelName(trigger.Name, 40),
		TitleStyle: chart.Style{
			Show:        true,
			Font:        plot.theme.font,
			FontColor:   chart.ColorAlternateGray,
			FillColor:   drawing.ColorFromHex(plot.theme.bgColor),
			StrokeColor: drawing.ColorFromHex(plot.theme.bgColor),
		},

		Width:  plot.width,
		Height: plot.height,

		Canvas: chart.Style{
			FillColor: drawing.ColorFromHex(plot.theme.bgColor),
		},
		Background: chart.Style{
			FillColor: drawing.ColorFromHex(plot.theme.bgColor),
			Padding:   bgPadding,
		},

		XAxis: chart.XAxis{
			Style: chart.Style{
				Show:        true,
				Font:        plot.theme.font,
				FontSize:    8,
				FontColor:   chart.ColorAlternateGray,
				StrokeColor: drawing.ColorFromHex(plot.theme.bgColor),
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
				Max:        plotLimits.highest,
				Min:        0,
			},
		},

		YAxisSecondary: chart.YAxis{
			ValueFormatter: yAxisValuesFormatter,
			Style: chart.Style{
				Show:        true,
				Font:        plot.theme.font,
				FontColor:   chart.ColorAlternateGray,
				StrokeColor: drawing.ColorFromHex(plot.theme.bgColor),
			},
			GridMinorStyle: gridStyle,
			GridMajorStyle: gridStyle,

			Range: &chart.ContinuousRange{
				Max: plotLimits.highest,
				Min: plotLimits.lowest,
			},
		},

		Series: plotSeries,
	}

	renderable.Elements = []chart.Renderable{
		getPlotLegend(&renderable, plot.width),
	}

	return renderable
}
