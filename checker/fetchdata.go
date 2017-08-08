package checker

import (
	"github.com/go-graphite/carbonapi/expr"
	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
	"github.com/moira-alert/moira-alert"
)

func FetchData(database moira.Database, pattern expr.MetricRequest, allowRealTimeAlerting bool) ([]*expr.MetricData, error) {
	metrics, err := database.GetPatternMetrics(pattern.Metric)
	if err != nil {
		return nil, err
	}

	metricDatas := make([]*expr.MetricData, 0)

	if len(metrics) == 0 {
		t := pb.FetchResponse{
			Name:      pattern.Metric,
			StartTime: pattern.From,
			StopTime:  pattern.Until,
			StepTime:  60,
			Values:    make([]float64, 0),
		}
		metricData := expr.MetricData{FetchResponse: t}
		metricDatas = append(metricDatas, &metricData)

	} else {
		firstMetric := metrics[0]
		retention, err := database.GetMetricRetention(firstMetric)
		if err != nil {
			return nil, err
		}
		datalist, err := database.GetMetricsValues(metrics, int64(pattern.From), int64(pattern.Until))
		valuesMap := unpackMetricsValues(datalist, retention, int64(pattern.From), int64(pattern.Until), allowRealTimeAlerting)
		for _, metric := range metrics {
			fetchResponse := pb.FetchResponse{
				Name:      metric,
				StartTime: pattern.From,
				StopTime:  pattern.Until,
				StepTime:  int32(retention),
				Values:    valuesMap[metric],
				IsAbsent:  make([]bool, len(valuesMap[metric]), len(valuesMap[metric])),
			}
			metricData := expr.MetricData{FetchResponse: fetchResponse}
			metricDatas = append(metricDatas, &metricData)
		}
	}
	return metricDatas, nil
}

func unpackMetricsValues(metricsData map[string][]*moira.MetricValue, retention int32, from int64, until int64, allowRealTimeAlerting bool) map[string][]float64 {
	getTimeSlot := func(timestamp int64) int64 {
		return (timestamp - from) / int64(retention)
	}

	valuesMap := make(map[string][]float64)
	for metric, metricData := range metricsData {
		points := make(map[int64]float64)
		for _, metricValue := range metricData {
			points[getTimeSlot(metricValue.Timestamp)] = metricValue.Value
		}

		lastTimeSlot := getTimeSlot(until)

		values := make([]float64, 0)
		//note that right boundary is exclusive
		for timeSlot := int64(0); timeSlot < lastTimeSlot; timeSlot++ {
			val, ok := points[timeSlot]
			if ok {
				values = append(values, val)
			}
		}

		lastPoint, ok := points[lastTimeSlot]
		if allowRealTimeAlerting && ok {
			values = append(values, lastPoint)
		}

		valuesMap[metric] = values
	}
	return valuesMap
}
