package local

import (
	"math"

	"github.com/go-graphite/carbonapi/expr/tags"
	"github.com/go-graphite/carbonapi/expr/types"
	pb "github.com/go-graphite/protocol/carbonapi_v3_pb"
	"github.com/moira-alert/moira"
)

const DefaultRetention = 60

// Context for metric fetch operation
type fetchData struct {
	database moira.Database
}

// Result of a pattern prefetch, when only names and retention are fetched
type metricsWithRetention struct {
	retention int64
	metrics   []string
}

func (fd *fetchData) fetchMetricNames(pattern string) (*metricsWithRetention, error) {
	metrics, err := fd.database.GetPatternMetrics(pattern)
	if err != nil {
		return nil, err
	}

	if len(metrics) == 0 {
		return &metricsWithRetention{retention: DefaultRetention, metrics: metrics}, nil
	}

	retention, err := fd.database.GetMetricRetention(metrics[0])
	if err != nil {
		return nil, err
	}

	return &metricsWithRetention{retention, metrics}, nil
}

func (fd *fetchData) fetchMetricValues(pattern string, metrics *metricsWithRetention, timer Timer) ([]*types.MetricData, error) {
	if len(metrics.metrics) == 0 {
		return fetchDataNoMetrics(timer, pattern), nil
	}

	dataList, err := fd.database.GetMetricsValues(metrics.metrics, timer.startTime, timer.stopTime-1)
	if err != nil {
		return nil, err
	}

	valuesMap := unpackMetricsValues(dataList, timer)

	metricsData := make([]*types.MetricData, 0, len(metrics.metrics))
	for _, metric := range metrics.metrics {
		metricsData = append(metricsData, createMetricData(metric, timer, valuesMap[metric]))
	}

	return metricsData, nil
}

func fetchDataNoMetrics(timer Timer, pattern string) []*types.MetricData {
	dataList := map[string][]*moira.MetricValue{pattern: make([]*moira.MetricValue, 0)}
	valuesMap := unpackMetricsValues(dataList, timer)
	metricsData := createMetricData(pattern, timer, valuesMap[pattern])

	return []*types.MetricData{metricsData}
}

func createMetricData(metric string, timer Timer, values []float64) *types.MetricData {
	fetchResponse := pb.FetchResponse{
		Name:      metric,
		StartTime: timer.startTime,
		StopTime:  timer.stopTime,
		StepTime:  timer.stepTime,
		Values:    values,
	}
	return &types.MetricData{FetchResponse: fetchResponse, Tags: tags.ExtractTags(metric)}
}

func unpackMetricsValues(metricsData map[string][]*moira.MetricValue, timer Timer) map[string][]float64 {
	valuesMap := make(map[string][]float64, len(metricsData))
	for metric, metricData := range metricsData {
		valuesMap[metric] = unpackMetricValues(metricData, timer)
	}
	return valuesMap
}

func unpackMetricValues(metricData []*moira.MetricValue, timer Timer) []float64 {
	points := make(map[int]*moira.MetricValue, len(metricData))
	for _, metricValue := range metricData {
		points[timer.GetTimeSlot(metricValue.RetentionTimestamp)] = metricValue
	}

	numberOfTimeSlots := timer.NumberOfTimeSlots()

	values := make([]float64, 0, numberOfTimeSlots)

	// note that right boundary is exclusive
	for timeSlot := 0; timeSlot < numberOfTimeSlots; timeSlot++ {
		val, ok := points[timeSlot]
		values = append(values, getMathFloat64(val, ok))
	}

	return values
}

func getMathFloat64(val *moira.MetricValue, ok bool) float64 {
	if ok {
		return val.Value
	}
	return math.NaN()
}
