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
func ResolveLimits(metricsData []*types.MetricData, from int32, to int32) Limits {
	allValues := make([]float64, 0)
	for _, metricData := range metricsData {
		for _, metricValue := range metricData.Values {
			if !math.IsNaN(metricValue) {
				allValues = append(allValues, metricValue)
			}
		}
	}
	lowest, highest := util.Math.MinAndMax(allValues...)
	return Limits{
		From:    Int32ToTime(from),
		To:      Int32ToTime(to),
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
