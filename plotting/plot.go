package plotting

import (
	"fmt"
	"time"

	"github.com/moira-alert/go-chart"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
)

const (
	plotNameLen = 40
)

// ErrNoPointsToRender is used to prevent unnecessary render calls.
type ErrNoPointsToRender struct {
	triggerID string
}

// ErrNoPointsToRender implementation with detailed error message.
func (err ErrNoPointsToRender) Error() string {
	return fmt.Sprintf("no points found to render trigger: %s", err.triggerID)
}

type PlotConfig struct {
	Width             int
	Height            int
	YAxisSecondaryCfg YAxisSecondaryConfig
}

type YAxisSecondaryConfig struct {
	EnablePrettyTicks bool
}

// Plot represents plot structure to render.
type Plot struct {
	cfg      PlotConfig
	theme    moira.PlotTheme
	location *time.Location
}

// GetPlotTemplate returns plot template.
func GetPlotTemplate(cfg PlotConfig, theme string, location *time.Location) (*Plot, error) {
	plotTheme, err := getPlotTheme(theme)
	if err != nil {
		return nil, err
	}

	if location == nil {
		return nil, fmt.Errorf("location not specified")
	}

	return &Plot{
		cfg:      cfg,
		theme:    plotTheme,
		location: location,
	}, nil
}

// GetRenderable returns go-chart to render.
func (plot *Plot) GetRenderable(targetName string, trigger *moira.Trigger, metricsData []metricSource.MetricData) (chart.Chart, error) {
	var renderable chart.Chart

	limits := resolveLimits(metricsData)

	curveSeriesList := getCurveSeriesList(metricsData, plot.theme)
	if len(curveSeriesList) == 0 {
		return renderable, ErrNoPointsToRender{triggerID: trigger.ID}
	}

	plotSeries := make([]chart.Series, 0, len(curveSeriesList))

	for _, curveSeries := range curveSeriesList {
		plotSeries = append(plotSeries, curveSeries)
	}

	thresholdSeriesList := getThresholdSeriesList(trigger, plot.theme, limits)
	plotSeries = append(plotSeries, thresholdSeriesList...)

	gridStyle := plot.theme.GetGridStyle()

	yAxisValuesFormatter, maxMarkLen := getYAxisValuesFormatter(limits)
	yAxisRange := limits.getThresholdAxisRange(trigger.TriggerType)

	name := fmt.Sprintf("%s - %s", targetName, trigger.Name)
	renderable = chart.Chart{
		Title:      sanitizeLabelName(name, plotNameLen),
		TitleStyle: plot.theme.GetTitleStyle(),

		Width:  plot.cfg.Width,
		Height: plot.cfg.Height,

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
			EnablePrettyTicks: plot.cfg.YAxisSecondaryCfg.EnablePrettyTicks,
		},

		Series: plotSeries,
	}

	renderable.Elements = []chart.Renderable{
		getPlotLegend(&renderable, plot.theme.GetLegendStyle(), plot.cfg.Width),
	}

	return renderable, nil
}
