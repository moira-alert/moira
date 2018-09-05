package target

import (
	"fmt"
	"runtime/debug"

	"github.com/go-graphite/carbonapi/expr"
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/go-graphite/carbonapi/pkg/parser"

	"github.com/moira-alert/moira"
)

// EvaluationResult represents evaluation target result and contains TimeSeries list, Pattern list and metric lists appropriate to given target
type EvaluationResult struct {
	TimeSeries []*TimeSeries
	Patterns   []string
	Metrics    []string
}

// EvaluateTarget is analogue of evaluateTarget method in graphite-web, that gets target metrics value from DB and Evaluate it using carbon-api eval package
func EvaluateTarget(database moira.Database, target string, from int64, until int64, allowRealTimeAlerting bool) (*EvaluationResult, error) {
	result := &EvaluationResult{
		TimeSeries: make([]*TimeSeries, 0),
		Patterns:   make([]string, 0),
		Metrics:    make([]string, 0),
	}

	targets := []string{target}
	targetIdx := 0
	for targetIdx < len(targets) {
		target := targets[targetIdx]
		targetIdx++
		expr2, _, err := parser.ParseExpr(target)
		if err != nil {
			return nil, ErrParseExpr{
				internalError: err,
				target:        target,
			}
		}
		patterns := expr2.Metrics()
		metricsMap, metrics, err := getPatternsMetricData(database, patterns, from, until, allowRealTimeAlerting)
		if err != nil {
			return nil, err
		}
		rewritten, newTargets, err := expr.RewriteExpr(expr2, from, until, metricsMap)
		if err != nil && err != parser.ErrSeriesDoesNotExist {
			return nil, fmt.Errorf("failed RewriteExpr: %s", err.Error())
		} else if rewritten {
			targets = append(targets, newTargets...)
		} else {
			metricDatas, err := func() (result []*types.MetricData, err error) {
				defer func() {
					if r := recover(); r != nil {
						result = nil
						err = fmt.Errorf("panic while evaluate target %s: message: '%s' stack: %s", target, r, debug.Stack())
					}
				}()
				result, err = expr.EvalExpr(expr2, from, until, metricsMap)
				if err != nil {
					if err == parser.ErrSeriesDoesNotExist {
						err = nil
					} else if isErrUnknownFunction(err) {
						err = ErrorUnknownFunction(err)
					} else {
						err = ErrEvalExpr{
							target:        target,
							internalError: err,
						}
					}
				}
				return result, err
			}()
			if err != nil {
				return nil, err
			}
			for _, metricData := range metricDatas {
				timeSeries := TimeSeries{
					MetricData: *metricData,
					Wildcard:   len(metrics) == 0,
				}
				result.TimeSeries = append(result.TimeSeries, &timeSeries)
			}
			result.Metrics = append(result.Metrics, metrics...)
			for _, pattern := range patterns {
				result.Patterns = append(result.Patterns, pattern.Metric)
			}
		}
	}
	return result, nil
}

func getPatternsMetricData(database moira.Database, patterns []parser.MetricRequest, from int64, until int64, allowRealTimeAlerting bool) (map[parser.MetricRequest][]*types.MetricData, []string, error) {
	metrics := make([]string, 0)
	metricsMap := make(map[parser.MetricRequest][]*types.MetricData)
	for _, pattern := range patterns {
		pattern.From += from
		pattern.Until += until
		metricsData, patternMetrics, err := FetchData(database, pattern.Metric, pattern.From, pattern.Until, allowRealTimeAlerting)
		if err != nil {
			return nil, nil, err
		}
		metricsMap[pattern] = metricsData
		metrics = append(metrics, patternMetrics...)
	}
	return metricsMap, metrics, nil
}
