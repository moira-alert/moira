package target

import (
	"math"

	"github.com/go-graphite/carbonapi/expr"
	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
	"github.com/moira-alert/moira"
)

// FetchData gets values of given pattern metrics from given interval and returns values and all found pattern metrics
func FetchData(database moira.Database, pattern string, from int64, until int64, allowRealTimeAlerting bool) ([]*expr.MetricData, []string, error) {
	metrics, err := database.GetPatternMetrics(pattern)
	if err != nil {
		return nil, nil, err
	}
	metricDatas := make([]*expr.MetricData, 0)

	if len(metrics) > 0 {
		firstMetric := metrics[0]
		retention, err := database.GetMetricRetention(firstMetric)
		if err != nil {
			return nil, nil, err
		}
		dataList, err := database.GetMetricsValues(metrics, from, until)
		if err != nil {
			return nil, nil, err
		}
		valuesMap := unpackMetricsValues(dataList, retention, from, until, allowRealTimeAlerting)
		for _, metric := range metrics {
			metricDatas = append(metricDatas, createMetricData(metric, from, until, retention, valuesMap[metric]))
		}
	} else {
		dataList := map[string][]*moira.MetricValue{pattern: make([]*moira.MetricValue, 0)}
		valuesMap := unpackMetricsValues(dataList, 60, from, until, allowRealTimeAlerting)
		metricDatas = append(metricDatas, createMetricData(pattern, from, until, 60, valuesMap[pattern]))
	}
	return metricDatas, metrics, nil
}

func createMetricData(metric string, from int64, until int64, retention int64, values []float64) *expr.MetricData {
	fetchResponse := pb.FetchResponse{
		Name:      metric,
		StartTime: int32(from),
		StopTime:  int32(until),
		StepTime:  int32(retention),
		Values:    values,
		IsAbsent:  getIsAbsent(values),
	}
	return &expr.MetricData{FetchResponse: fetchResponse}
}
func getIsAbsent(values []float64) []bool {
	isAbsent := make([]bool, len(values))
	for i, value := range values {
		if math.IsNaN(value) {
			isAbsent[i] = true
		}
	}
	return isAbsent
}

func unpackMetricsValues(metricsData map[string][]*moira.MetricValue, retention int64, from int64, until int64, allowRealTimeAlerting bool) map[string][]float64 {
	retentionFrom := roundToMinimalHighestRetention(from, retention)
	getTimeSlot := func(timestamp int64) int64 {
		return (timestamp - retentionFrom) / retention
	}

	valuesMap := make(map[string][]float64, len(metricsData))
	for metric, metricData := range metricsData {
		valuesMap[metric] = unpackMetricValues(metricData, until, allowRealTimeAlerting, getTimeSlot)
	}
	return valuesMap
}

func unpackMetricValues(metricData []*moira.MetricValue, until int64, allowRealTimeAlerting bool, getTimeSlot func(int64) int64) []float64 {
	points := make(map[int64]*moira.MetricValue, len(metricData))
	for _, metricValue := range metricData {
		points[getTimeSlot(metricValue.RetentionTimestamp)] = metricValue
	}

	lastTimeSlot := getTimeSlot(until)

	values := make([]float64, 0, lastTimeSlot+1)
	// note that right boundary is exclusive
	for timeSlot := int64(0); timeSlot < lastTimeSlot; timeSlot++ {
		val, ok := points[timeSlot]
		values = append(values, getMathFloat64(val, ok))
	}

	lastPoint, ok := points[lastTimeSlot]
	if allowRealTimeAlerting && ok {
		values = append(values, getMathFloat64(lastPoint, ok))
	}
	return values
}

func getMathFloat64(val *moira.MetricValue, ok bool) float64 {
	if ok {
		return val.Value
	}
	return math.NaN()
}

func roundToMinimalHighestRetention(ts, retention int64) int64 {
	if (ts % retention) == 0 {
		return ts
	}
	return (ts + retention) / retention * retention
}
