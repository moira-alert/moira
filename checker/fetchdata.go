package checker

import (
	"github.com/go-graphite/carbonapi/expr"
	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
	"github.com/moira-alert/moira-alert"
	"math"
)

func FetchData(database moira.Database, pattern expr.MetricRequest, allowRealTimeAlerting bool) ([]*expr.MetricData, []string, error) {
	metrics, err := database.GetPatternMetrics(pattern.Metric)
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
		dataList, err := database.GetMetricsValues(metrics, int64(pattern.From), int64(pattern.Until))
		if err != nil {
			return nil, nil, err
		}
		valuesMap := unpackMetricsValues(dataList, retention, int64(pattern.From), int64(pattern.Until), allowRealTimeAlerting)
		for _, metric := range metrics {
			metricDatas = append(metricDatas, createMetricData(metric, pattern.From, pattern.Until, int32(retention), valuesMap[metric]))
		}
	}

	return metricDatas, metrics, nil
}

func createMetricData(metric string, from int32, until int32, retention int32, values []float64) *expr.MetricData {
	fetchResponse := pb.FetchResponse{
		Name:      metric,
		StartTime: from,
		StopTime:  until,
		StepTime:  retention,
		Values:    values,
		IsAbsent:  make([]bool, len(values), len(values)),
	}
	return &expr.MetricData{FetchResponse: fetchResponse}
}

func unpackMetricsValues(metricsData map[string][]*moira.MetricValue, retention int32, from int64, until int64, allowRealTimeAlerting bool) map[string][]float64 {
	retentionFrom := roundToMinimalHighestRetention(from, retention)
	getTimeSlot := func(timestamp int64) int64 {
		return (timestamp - retentionFrom) / int64(retention)
	}

	valuesMap := make(map[string][]float64)
	for metric, metricData := range metricsData {
		points := make(map[int64]float64)
		for _, metricValue := range metricData {
			points[getTimeSlot(metricValue.RetentionTimestamp)] = metricValue.Value
		}

		lastTimeSlot := getTimeSlot(until)

		values := make([]float64, 0)
		//note that right boundary is exclusive
		for timeSlot := int64(0); timeSlot < lastTimeSlot; timeSlot++ {
			val, ok := points[timeSlot]
			values = append(values, getMathFloat64(val, ok))
		}

		lastPoint, ok := points[lastTimeSlot]
		if allowRealTimeAlerting && ok {
			values = append(values, getMathFloat64(lastPoint, ok))
		}

		valuesMap[metric] = values
	}
	return valuesMap
}

func getMathFloat64(val float64, ok bool) float64 {
	if ok {
		return val
	} else {
		return math.NaN()
	}
}

func roundToMinimalHighestRetention(ts int64, retention int32) int64 {
	ret := int64(retention)
	if (ts % ret) == 0 {
		return ts
	}
	return (ts + ret) / ret * ret
}
