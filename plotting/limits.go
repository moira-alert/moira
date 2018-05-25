package plotting

import (
	"time"
	"math"

	"github.com/go-graphite/carbonapi/expr/types"
	//"github.com/wcharczuk/go-chart/util"
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
	var lowest float64
	var highest float64
	for _, metricData := range metricsData {
		startInd, _ := ResolveFirstPoint(metricData)
		for metricValInd := startInd; metricValInd < len(metricData.Values); metricValInd++ {
			metricVal := metricData.Values[metricValInd]
			if metricValInd == startInd {
				lowest = metricVal
				highest = metricVal
			}
			if !math.IsNaN(metricVal) {
				lowest = math.Min(lowest, metricVal)
				highest = math.Max(highest, metricVal)
				//lowest, highest = util.Math.MinAndMax(lowest, highest, metricVal)
			}
		}
	}
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
