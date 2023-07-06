package prometheus

import (
	"sort"
	"strings"

	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/prometheus/common/model"
)

func convertToFetchResult(mat model.Matrix) *FetchResult {
	result := FetchResult{
		MetricsData: make([]metricSource.MetricData, 0, len(mat)),
	}

	for _, res := range mat {
		values := []float64{}
		for _, v := range res.Values {
			values = append(values, float64(v.Value))
		}

		data := metricSource.MetricData{
			Name:      targetFromTags(res.Metric),
			StartTime: res.Values[0].Timestamp.Unix(),
			StopTime:  res.Values[len(res.Values)-1].Timestamp.Unix(),
			StepTime:  StepTimeSeconds,
			Values:    values,
			Wildcard:  false,
		}
		result.MetricsData = append(result.MetricsData, data)
	}

	return &result
}

func targetFromTags(tags model.Metric) string {
	if len(tags) == 1 {
		for _, value := range tags {
			return string(value)
		}
	}

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
		key, value := tag.key, tag.value

		if key == "__name__" {
			continue
		}
		if target.Len() != 0 {
			target.WriteRune(';')
		}
		target.WriteString(key)
		target.WriteRune('=')
		target.WriteString(value)
	}

	return target.String()
}
