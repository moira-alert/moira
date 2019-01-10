package plotting

import (
	"fmt"
	"time"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart"

	"github.com/moira-alert/moira"
)

// ErrNoPointsToRender is used to prevent unnecessary render calls
type ErrNoPointsToRender struct {
	triggerID string
}

// ErrNoPointsToRender implementation with detailed error message
func (err ErrNoPointsToRender) Error() string {
	return fmt.Sprintf("no points found to render trigger: %s", err.triggerID)
}

// Plot represents plot structure to render
type Plot struct {
	theme    moira.PlotTheme
	location *time.Location
	width    int
	height   int
}

// GetPlotTemplate returns plot template
func GetPlotTemplate(theme string, location *time.Location) (*Plot, error) {
	plotTheme, err := getPlotTheme(theme)
	if err != nil {
		return nil, err
	}
	if location == nil {
		return nil, fmt.Errorf("location not specified")
	}
	return &Plot{
		theme:    plotTheme,
		location: location,
		width:    800,
		height:   400,
	}, nil
}

// GetRenderable returns go-chart to render
func (plot *Plot) GetRenderable(trigger *moira.Trigger, metricsData []*types.MetricData, metricsWhitelist []string) (chart.Chart, error) {
	var renderable chart.Chart

	plotSeries := make([]chart.Series, 0)

	metricsData = toLimitedMetricsData(metricsData, metricsWhitelist)
	limits := resolveLimits(metricsData)

	curveSeriesList := getCurveSeriesList(metricsData, plot.theme)
	if len(curveSeriesList) == 0 {
		return renderable, ErrNoPointsToRender{triggerID: trigger.ID}
	}

	for _, curveSeries := range curveSeriesList {
		plotSeries = append(plotSeries, curveSeries)
	}

	thresholdSeriesList := getThresholdSeriesList(trigger, plot.theme, limits)
	plotSeries = append(plotSeries, thresholdSeriesList...)

	gridStyle := plot.theme.GetGridStyle()

	yAxisValuesFormatter, maxMarkLen := getYAxisValuesFormatter(limits)
	yAxisRange := limits.getThresholdAxisRange(trigger.TriggerType)

	renderable = chart.Chart{

		Title:      sanitizeLabelName(trigger.Name, 40),
		TitleStyle: plot.theme.GetTitleStyle(),

		Width:  plot.width,
		Height: plot.height,

		Canvas:     plot.theme.GetCanvasStyle(),
		Background: plot.theme.GetBackgroundStyle(maxMarkLen),

		XAxis: chart.XAxis{
			Style:          plot.theme.GetXAxisStyle(),
			GridMinorStyle: gridStyle,
			GridMajorStyle: gridStyle,
			ValueFormatter: getTimeValueFormatter(plot.location, "15:04"),
			Range: &chart.ContinuousRange{
				Min: float64(limits.from.UnixNano()),
				Max: float64(limits.to.UnixNano()),
			},
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

	return renderable, nil
}
