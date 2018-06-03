package plotting

import (
	"math"
	"time"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart/util"
)

// Limits is a set of limits for given metricsData
type Limits struct {
	From    time.Time
	To      time.Time
	Lowest  float64
	Highest float64
}

// ResolveLimits returns common plot limits
func ResolveLimits(metricsData []*types.MetricData) Limits {
	allValues := make([]float64, 0)
	allTimes := make([]time.Time, 0)
	for _, metricData := range metricsData {
		for _, metricValue := range metricData.Values {
			if !math.IsNaN(metricValue) {
				allValues = append(allValues, metricValue)
			}
		}
		allTimes = append(allTimes, Int32ToTime(metricData.StartTime))
		allTimes = append(allTimes, Int32ToTime(metricData.StopTime))
	}
	from, to := util.Math.MinAndMaxOfTime(allTimes...)
	lowest, highest := util.Math.MinAndMax(allValues...)
	if lowest == highest {
		lowest = lowest + 5
		highest = highest + 5
	}
	return Limits{
		From:    from,
		To:      to,
		Lowest:  lowest,
		Highest: highest,
	}
}

// FormsSetContaining returns true if dot can belong to a set formed from limit values
func (limits Limits) FormsSetContaining(dot float64) bool {
	if (dot >= limits.Lowest) && (dot <= limits.Highest) {
		return true
	}
	return false
}
