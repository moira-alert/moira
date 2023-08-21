package prometheus

import (
	"sort"
	"strings"

	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/prometheus/common/model"
)

func convertToFetchResult(mat model.Matrix, from, until int64) *FetchResult {
	result := FetchResult{
		MetricsData: make([]metricSource.MetricData, 0, len(mat)),
	}

	for _, res := range mat {
		values := []float64{}
		for _, v := range res.Values {
			values = append(values, float64(v.Value))
		}

		start, stop := from, until
		if len(res.Values) != 0 {
			start = res.Values[0].Timestamp.Unix()
			stop = res.Values[len(res.Values)-1].Timestamp.Unix()
		}

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
