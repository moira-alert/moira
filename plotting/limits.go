package plotting

import (
	"math"
	"time"

	"github.com/beevee/go-chart"
	"github.com/beevee/go-chart/util"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
)

const (
	// defaultRangeDelta is an additional value to
	// cover cases with equal highest/lowest limits values
	defaultRangeDelta = 10
	// defaultYAxisRangePercent is a default percent value to
	// generate plotLimits lowest/highest additional increment
	// used in plot-prettifying purposes only
	defaultYAxisRangePercent = 10
)

// plotLimits is a set of limits for given metricsData
type plotLimits struct {
	from    time.Time
	to      time.Time
	lowest  float64
	highest float64
}

// resolveLimits returns common plot limits
func resolveLimits(metricsData []metricSource.MetricData) plotLimits {
	allValues := make([]float64, 0)
	allTimes := make([]time.Time, 0)
	for _, metricData := range metricsData {
		for _, metricValue := range metricData.Values {
			if !math.IsNaN(metricValue) {
				allValues = append(allValues, metricValue)
			}
		}
		allTimes = append(allTimes, moira.Int64ToTime(metricData.StartTime))
		allTimes = append(allTimes, moira.Int64ToTime(metricData.StopTime))
	}
	from, to := util.Time.StartAndEnd(allTimes...)
	lowest, highest := util.Math.MinAndMax(allValues...)
	if highest == lowest {
		highest = highest + (defaultRangeDelta / 2)
		lowest = lowest - (defaultRangeDelta / 2)
	}
	yAxisIncrement := percentsOfRange(lowest, highest, defaultYAxisRangePercent)
	if highest > 0 {
		highest = highest + yAxisIncrement
	}
	if lowest < 0 {
		lowest = lowest - yAxisIncrement
	}
	return plotLimits{
		from:    from,
		to:      to,
		lowest:  lowest,
		highest: highest,
	}
}

// getThresholdAxisRange returns threshold axis range
func (limits *plotLimits) getThresholdAxisRange(triggerType string) chart.ContinuousRange {
	if triggerType == moira.RisingTrigger {
		return chart.ContinuousRange{
			Descending: true,
			Max:        limits.highest - limits.lowest,
			Min:        0,
		}
	}
	return chart.ContinuousRange{
		Descending: false,
		Max:        limits.highest,
		Min:        limits.lowest,
	}
}

// formsSetContaining returns true if dot can belong to a set formed from limit values
func (limits plotLimits) formsSetContaining(dot float64) bool {
	if (dot >= limits.lowest) && (dot <= limits.highest) {
		return true
	}
	return false
}
