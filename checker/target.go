package checker

import (
	"github.com/go-graphite/carbonapi/expr"
	"github.com/moira-alert/moira-alert"
)

func EvaluateTarget(database moira.Database, target string, from int64, until int64, allowRealTimeAlerting bool) ([]*expr.MetricData, error) {
	expr2, _, err := expr.ParseExpr(target)
	if err != nil {
		return nil, err
	}
	metrics := expr2.Metrics()
	metricsMap := make(map[expr.MetricRequest][]*expr.MetricData, 0)
	for _, metric := range metrics {
		metric.From += int32(from)
		metric.Until += int32(until)
		metricDatas, err := FetchData(database, metric, allowRealTimeAlerting)
		if err != nil {
			return nil, err
		}
		metricsMap[metric] = metricDatas
	}
	return expr.EvalExpr(expr2, int32(from), int32(until), metricsMap)
}
