package prometheus

import (
	"sort"
	"strings"

	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/prometheus/common/model"
)

func convertToFetchResult(mat model.Matrix, from, until int64, allowRealTimeAlerting bool) *FetchResult {
	result := FetchResult{
		MetricsData: make([]metricSource.MetricData, 0, len(mat)),
	}

	for _, res := range mat {
		resValues := TrimValuesIfNescesary(res.Values, allowRealTimeAlerting)

		values := make([]float64, 0, len(resValues))
		for _, v := range resValues {
			values = append(values, float64(v.Value))
		}

		start, stop := StartStopFromValues(resValues, from, until)
		data := metricSource.MetricData{
			Name:      targetFromTags(res.Metric),
			StartTime: start,
			StopTime:  stop,
			StepTime:  StepTimeSeconds,
			Values:    values,
			Wildcard:  false,
		}
		result.MetricsData = append(result.MetricsData, data)
	}

	return &result
}

func StartStopFromValues(values []model.SamplePair, from, until int64) (int64, int64) {
	start, stop := from, until
	if len(values) != 0 {
		start = values[0].Timestamp.Unix()
		stop = values[len(values)-1].Timestamp.Unix()
	}
	return start, stop
}

func TrimValuesIfNescesary(values []model.SamplePair, allowRealTimeAlerting bool) []model.SamplePair {
	if allowRealTimeAlerting || len(values) == 0 {
		return values
	}

	return values[:len(values)-1]
}

func targetFromTags(tags model.Metric) string {
	target := strings.Builder{}
	if name, ok := tags["__name__"]; ok {
		target.WriteString(string(name))
	}

	tagsList := make([]struct{ key, value string }, 0)
	for key, value := range tags {
		tagsList = append(tagsList, struct{ key, value string }{string(key), string(value)})
	}

	sort.Slice(tagsList, func(i, j int) bool {
		a, b := tagsList[i], tagsList[j]
		if a.key != b.key {
			return a.key < b.key
		}

		return a.value < b.value
	})

	for _, tag := range tagsList {
		if tag.key == "__name__" {
			continue
		}
		if target.Len() != 0 {
			target.WriteRune(';')
		}
		target.WriteString(tag.key)
		target.WriteRune('=')
		target.WriteString(tag.value)
	}

	return target.String()
}
