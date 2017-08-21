package target

import (
	"github.com/go-graphite/carbonapi/expr"
	"math"
)

type TimeSeries expr.MetricData

func (timeSeries *TimeSeries) GetTimestampValue(valueTimestamp int64) float64 {
	if valueTimestamp < int64(timeSeries.StartTime) {
		return math.NaN()
	}
	valueIndex := int((valueTimestamp - int64(timeSeries.StartTime)) / int64(timeSeries.StepTime))
	if len(timeSeries.IsAbsent) > valueIndex && timeSeries.IsAbsent[valueIndex] {
		return math.NaN()
	}
	if len(timeSeries.Values) <= valueIndex {
		return math.NaN()
	}
	return timeSeries.Values[valueIndex]
}
