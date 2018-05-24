package plotting

import (
	"math"
	"time"

	"github.com/go-graphite/carbonapi/expr/types"
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
	// TODO: Refactor to not to use metricsData[0]
	// TODO: this method must be allowed to use empty float arrays
	from := float64(metricsData[0].StartTime)
	to := float64(metricsData[0].StopTime)
	lowest := float64(metricsData[0].Values[0])
	highest := lowest
	for _, metricData := range metricsData {
		from = math.Min(float64(metricData.StartTime), from)
		to = math.Max(float64(metricData.StopTime), to)
		for _, metricVal := range metricData.Values {
			lowest = math.Min(metricVal, lowest)
			highest = math.Max(metricVal, highest)
		}
	}
	return Limits{
		From:    Int32ToTime(int32(from)),
		To:      Int32ToTime(int32(to)),
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
