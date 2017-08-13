package checker

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/go-graphite/carbonapi/expr"
	"github.com/moira-alert/moira-alert"
)

var ErrEvaluateTarget = errors.New("Invalid graphite targets")

func EvaluateTarget(database moira.Database, target string, from int64, until int64, allowRealTimeAlerting bool) ([]*TimeSeries, []string, error) {
	targets := []string{target}
	targetIdx := 0
	results := make([]*TimeSeries, 0)
	allMetrics := make([]string, 0)
	for targetIdx < len(targets) {
		target := targets[targetIdx]
		targetIdx++
		expr2, _, err := expr.ParseExpr(target)
		if err != nil {
			return nil, nil, err
		}
		metricsMap, metrics, err := getPatternsMetricData(database, expr2.Metrics(), from, until, allowRealTimeAlerting)
		if err != nil {
			return nil, nil, err
		}
		rewritten, newTargets, err := expr.RewriteExpr(expr2, int32(from), int32(until), metricsMap)
		if err != nil && err != expr.ErrSeriesDoesNotExist {
			return nil, nil, fmt.Errorf("Failed RewriteExpr: %s", err.Error())
		} else if rewritten {
			targets = append(targets, newTargets...)
		} else {
			metricDatas, err := expr.EvalExpr(expr2, int32(from), int32(until), metricsMap)
			if err != nil && err != expr.ErrSeriesDoesNotExist {
				return nil, nil, ErrEvaluateTarget
			}
			for _, metricData := range metricDatas {
				var timeSeries TimeSeries = TimeSeries(*metricData)
				results = append(results, &timeSeries)
			}
			allMetrics = append(allMetrics, metrics...)
		}
	}
	return results, allMetrics, nil
}

func getPatternsMetricData(database moira.Database, patterns []expr.MetricRequest, from int64, until int64, allowRealTimeAlerting bool) (map[expr.MetricRequest][]*expr.MetricData, []string, error) {
	metrics := make([]string, 0)
	metricsMap := make(map[expr.MetricRequest][]*expr.MetricData, 0)
	for _, pattern := range patterns {
		pattern.From += int32(from)
		pattern.Until += int32(until)
		metricDatas, patternMetrics, err := FetchData(database, pattern.Metric, int64(pattern.From), int64(pattern.Until), allowRealTimeAlerting)
		if err != nil {
			return nil, nil, err
		}
		metricsMap[pattern] = metricDatas
		metrics = append(metrics, patternMetrics...)
	}
	return metricsMap, metrics, nil
}
