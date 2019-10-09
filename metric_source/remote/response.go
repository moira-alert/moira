package remote

import (
	"encoding/json"
	"math"

	metricSource "github.com/moira-alert/moira/metric_source"
)

type graphiteMetric struct {
	Target     string
	DataPoints [][2]*float64
}

func convertResponse(metricsData []metricSource.MetricData, allowRealTimeAlerting bool) FetchResult {
	if allowRealTimeAlerting {
		return FetchResult{MetricsData: metricsData}
	}

	result := make([]metricSource.MetricData, 0, len(metricsData))
	for _, metricData := range metricsData {
		// remove last value
		metricData.Values = metricData.Values[:len(metricData.Values)-1]
		result = append(result, metricData)
	}
	return FetchResult{MetricsData: result}
}

func decodeBody(body []byte) ([]metricSource.MetricData, error) {
	var tmp []graphiteMetric
	err := json.Unmarshal(body, &tmp)
	if err != nil {
		return nil, err
	}
	res := make([]metricSource.MetricData, 0, len(tmp))
	for _, m := range tmp {
		var stepTime int64 = 60
		if len(m.DataPoints) > 1 {
			stepTime = int64(*m.DataPoints[1][1] - *m.DataPoints[0][1])
		}
		metricData := metricSource.MetricData{
			Name:      m.Target,
			StartTime: int64(*m.DataPoints[0][1]),
			StopTime:  int64(*m.DataPoints[len(m.DataPoints)-1][1]),
			StepTime:  stepTime,
			Values:    make([]float64, len(m.DataPoints)),
		}
		for i, v := range m.DataPoints {
			if v[0] == nil {
				metricData.Values[i] = math.NaN()
			} else {
				metricData.Values[i] = *v[0]
			}
		}
		res = append(res, metricData)
	}
	return res, nil
}
