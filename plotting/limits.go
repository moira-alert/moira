package plotting

import (
	"math"
	"time"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart/util"
)

// plotLimits is a set of limits for given metricsData
type plotLimits struct {
	from    time.Time
	to      time.Time
	lowest  float64
	highest float64
}

// resolveLimits returns common plot limits
func resolveLimits(metricsData []*types.MetricData) plotLimits {
	allValues := make([]float64, 0)
	allTimes := make([]time.Time, 0)
	for _, metricData := range metricsData {
		for _, metricValue := range metricData.Values {
			if !math.IsNaN(metricValue) {
				allValues = append(allValues, metricValue)
			}
		}
		allTimes = append(allTimes, int64ToTime(metricData.StartTime))
		allTimes = append(allTimes, int64ToTime(metricData.StopTime))
	}
	from, to := util.Math.MinAndMaxOfTime(allTimes...)
	lowest, highest := util.Math.MinAndMax(allValues...)
	if lowest == highest {
		lowest = lowest + 5
		highest = highest + 5
	}
	return plotLimits{
		from:    from,
		to:      to,
		lowest:  lowest,
		highest: highest,
	}
}

// formsSetContaining returns true if dot can belong to a set formed from limit values
func (limits plotLimits) formsSetContaining(dot float64) bool {
	if (dot >= limits.lowest) && (dot <= limits.highest) {
		return true
	}
	return false
}
