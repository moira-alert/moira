package target

import (
	"fmt"
	"github.com/go-graphite/carbonapi/expr"
	"github.com/moira-alert/moira-alert"
)

//ErrEvaluateTarget represent evaluation error by carbon-api eval method
var ErrEvaluateTarget = fmt.Errorf("Invalid graphite targets")

//EvaluationResult represents evaluation target result and contains TimeSeries list, Pattern list and metric lists appropriate to given target
type EvaluationResult struct {
	TimeSeries []*TimeSeries
	Patterns   []string
	Metrics    []string
}

//EvaluateTarget is analogue of evaluateTarget method in graphite-web, that gets target metrics value from DB and Evaluate it using carbon-api eval package
func EvaluateTarget(database moira.Database, target string, from int64, until int64, allowRealTimeAlerting bool) (*EvaluationResult, error) {
	result := EvaluationResult{
		TimeSeries: make([]*TimeSeries, 0),
		Patterns:   make([]string, 0),
		Metrics:    make([]string, 0),
	}

	targets := []string{target}
	targetIdx := 0
	for targetIdx < len(targets) {
		target := targets[targetIdx]
		targetIdx++
		expr2, _, err := expr.ParseExpr(target)
		if err != nil {
			return nil, err
		}
		patterns := expr2.Metrics()
		metricsMap, metrics, err := getPatternsMetricData(database, patterns, from, until, allowRealTimeAlerting)
		if err != nil {
			return nil, err
		}
		rewritten, newTargets, err := expr.RewriteExpr(expr2, int32(from), int32(until), metricsMap)
		if err != nil && err != expr.ErrSeriesDoesNotExist {
			return nil, fmt.Errorf("Failed RewriteExpr: %s", err.Error())
		} else if rewritten {
			targets = append(targets, newTargets...)
		} else {
			metricDatas, err := expr.EvalExpr(expr2, int32(from), int32(until), metricsMap)
			if err != nil && err != expr.ErrSeriesDoesNotExist {
				return nil, ErrEvaluateTarget
			}
			for _, metricData := range metricDatas {
				var timeSeries TimeSeries = TimeSeries(*metricData)
				result.TimeSeries = append(result.TimeSeries, &timeSeries)
			}
			result.Metrics = append(result.Metrics, metrics...)
			for _, pattern := range patterns {
				result.Patterns = append(result.Patterns, pattern.Metric)
			}
		}
	}
	return &result, nil
}

func getPatternsMetricData(database moira.Database, patterns []expr.MetricRequest, from int64, until int64, allowRealTimeAlerting bool) (map[expr.MetricRequest][]*expr.MetricData, []string, error) {
	metrics := make([]string, 0)
	metricsMap := make(map[expr.MetricRequest][]*expr.MetricData)
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
