package plotting

import (
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart"

	"github.com/moira-alert/moira"
)

// Plot represents plot structure to render
type Plot struct {
	theme  moira.PlotTheme
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
func (plot *Plot) GetRenderable(trigger *moira.Trigger, metricsData []*types.MetricData, metricsWhitelist []string) chart.Chart {
	plotSeries := make([]chart.Series, 0)
	limits := resolveLimits(metricsData)

	curveSeriesList := getCurveSeriesList(metricsData, plot.theme, metricsWhitelist)
	for _, curveSeries := range curveSeriesList {
		plotSeries = append(plotSeries, curveSeries)
	}

	thresholdSeriesList := getThresholdSeriesList(trigger, plot.theme, limits)
	plotSeries = append(plotSeries, thresholdSeriesList...)

	gridStyle := plot.theme.GetGridStyle()

	yAxisValuesFormatter := getYAxisValuesFormatter(limits)
	yAxisRange := limits.getThresholdAxisRange(trigger.TriggerType)

	renderable := chart.Chart{

		Title:      sanitizeLabelName(trigger.Name, 40),
		TitleStyle: plot.theme.GetTitleStyle(),

		Width:  plot.width,
		Height: plot.height,

		Canvas:     plot.theme.GetCanvasStyle(),
		Background: plot.theme.GetBackgroundStyle(),

		XAxis: chart.XAxis{
			Style:          plot.theme.GetXAxisStyle(),
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
			Range:          &yAxisRange,
		},

		YAxisSecondary: chart.YAxis{
			ValueFormatter: yAxisValuesFormatter,
			Style:          plot.theme.GetYAxisStyle(),
			GridMinorStyle: gridStyle,
			GridMajorStyle: gridStyle,
			Range: &chart.ContinuousRange{
				Max: limits.highest,
				Min: limits.lowest,
			},
		},

		Series: plotSeries,
	}

	renderable.Elements = []chart.Renderable{
		getPlotLegend(&renderable, plot.theme.GetLegendStyle(), plot.width),
	}

	return renderable
}
