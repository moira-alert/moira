package metricSource

import "math"

// MetricData is moira implementation of target evaluation result
type MetricData struct {
	Name      string
	StartTime int64
	StopTime  int64
	StepTime  int64
	Values    []float64
	Wildcard  bool
}

// MakeMetricData creates new metrics data with given metric timeseries
func MakeMetricData(name string, values []float64, step, start int64) *MetricData {
	stop := start + int64(len(values))*step
	return &MetricData{
		Name:      name,
		Values:    values,
		StartTime: start,
		StepTime:  step,
		StopTime:  stop,
	}
}

// MakeEmptyMetricData create MetricData with given interval and retention step with all empty metric points
func MakeEmptyMetricData(name string, step, start, stop int64) *MetricData {
	values := make([]float64, 0)
	for i := start; i < stop; i += step {
		values = append(values, math.NaN())
	}
	return &MetricData{
		Name:      name,
		Values:    values,
		StartTime: start,
		StepTime:  step,
		StopTime:  stop,
	}
}

// GetTimestampValue gets value of given timestamp index, if value is Nil, then return NaN
func (timeSeries *MetricData) GetTimestampValue(valueTimestamp int64) float64 {
	if valueTimestamp < timeSeries.StartTime {
		return math.NaN()
	}
	valueIndex := int((valueTimestamp - timeSeries.StartTime) / timeSeries.StepTime)
	if len(timeSeries.Values) <= valueIndex {
		return math.NaN()
	}
	return timeSeries.Values[valueIndex]
}
