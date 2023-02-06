package local

import (
	"math"

	"github.com/go-graphite/carbonapi/expr/tags"
	"github.com/go-graphite/carbonapi/expr/types"
	pb "github.com/go-graphite/protocol/carbonapi_v3_pb"
	"github.com/moira-alert/moira"
)

// FetchData gets values of given pattern metrics from given interval and returns values and all found pattern metrics
func FetchData(database moira.Database, pattern string, from, until int64, allowRealTimeAlerting bool) ([]*types.MetricData, []string, error) {
	metrics, err := database.GetPatternMetrics(pattern)
	if err != nil {
		return nil, nil, err
	}

	if len(metrics) == 0 {
		timer := MakeTimer(from, until, 60, allowRealTimeAlerting)
		return fetchDataNoMetrics(timer, pattern), metrics, nil
	}

	ctx := &FetchDataCtx{
		database:              database,
		pattern:               pattern,
		from:                  from,
		until:                 until,
		metrics:               metrics,
		allowRealTimeAlerting: allowRealTimeAlerting,
	}

	metricsData, err := ctx.fetchDataWithMetrics()
	if err != nil {
		return nil, nil, err
	}
	return metricsData, metrics, nil

}

func fetchDataNoMetrics(timer Timer, pattern string) []*types.MetricData {
	dataList := map[string][]*moira.MetricValue{pattern: make([]*moira.MetricValue, 0)}
	valuesMap := unpackMetricsValues(dataList, timer)
	metricsData := createMetricData(pattern, timer, valuesMap[pattern])

	return []*types.MetricData{metricsData}
}

type FetchDataCtx struct {
	database              moira.Database
	pattern               string
	metrics               []string
	from                  int64
	until                 int64
	allowRealTimeAlerting bool
}

func (ctx *FetchDataCtx) fetchDataWithMetrics() ([]*types.MetricData, error) {
	timer, err := ctx.makeTimer()
	if err != nil {
		return nil, err
	}

	dataList, err := ctx.database.GetMetricsValues(ctx.metrics, ctx.from, ctx.until)
	if err != nil {
		return nil, err
	}

	valuesMap := unpackMetricsValues(dataList, timer)

	metricsData := make([]*types.MetricData, 0, len(ctx.metrics))
	for _, metric := range ctx.metrics {
		metricsData = append(metricsData, createMetricData(metric, timer, valuesMap[metric]))
	}

	return metricsData, nil
}

func (ctx *FetchDataCtx) makeTimer() (Timer, error) {
	firstMetric := ctx.metrics[0]
	retention, err := ctx.database.GetMetricRetention(firstMetric)
	if err != nil {
		return Timer{}, err
	}
	return MakeTimer(ctx.from, ctx.until, retention, ctx.allowRealTimeAlerting), nil
}

func createMetricData(metric string, timer Timer, values []float64) *types.MetricData {
	fetchResponse := pb.FetchResponse{
		Name:      metric,
		StartTime: timer.from,
		StopTime:  timer.until,
		StepTime:  timer.retention,
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

	// TODO: Handle allowRealTimeAlerting correctly
	lastPoint, ok := points[numberOfTimeSlots]
	if timer.allowRealTimeAlerting && ok {
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
