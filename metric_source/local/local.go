package local

import (
	"fmt"
	"runtime/debug"

	"github.com/go-graphite/carbonapi/expr"
	"github.com/go-graphite/carbonapi/expr/functions"
	"github.com/go-graphite/carbonapi/expr/rewrite"
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/go-graphite/carbonapi/pkg/parser"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
)

// Local is implementation of MetricSource interface, which implements fetch metrics method from moira database installation
type Local struct {
	dataBase moira.Database
}

// Create configures local metric source
func Create(dataBase moira.Database) metricSource.MetricSource {
	// configure carbon-api functions
	rewrite.New(make(map[string]string))
	functions.New(make(map[string]string))

	return &Local{
		dataBase: dataBase,
	}
}

// Fetch is analogue of evaluateTarget method in graphite-web, that gets target metrics value from DB and Evaluate it using carbon-api eval package
func (local *Local) Fetch(target string, from int64, until int64, allowRealTimeAlerting bool) (metricSource.FetchResult, error) {
	// Don't fetch intervals larger than metrics TTL to prevent OOM errors
	// See https://github.com/moira-alert/moira/pull/519
	from = moira.MaxInt64(from, until-local.dataBase.GetMetricsTTLSeconds())

	result := CreateEmptyFetchResult()

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
		metricsMap, metrics, err := getPatternsMetricData(local.dataBase, patterns, from, until, allowRealTimeAlerting)
		if err != nil {
			return nil, err
		}
		rewritten, newTargets, err := expr.RewriteExpr(expr2, from, until, metricsMap)
		if err != nil && err != parser.ErrSeriesDoesNotExist {
			return nil, fmt.Errorf("failed RewriteExpr: %s", err.Error())
		} else if rewritten {
			targets = append(targets, newTargets...)
		} else {
			metricsData, err := func() (result []*types.MetricData, err error) {
				defer func() {
					if r := recover(); r != nil {
						result = nil
						err = ErrEvaluateTargetFailedWithPanic{target: target, recoverMessage: r, stackRecord: debug.Stack()}
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
			for _, metricData := range metricsData {
				md := *metricData
				result.MetricsData = append(result.MetricsData, metricSource.MetricData{
					Name:      md.Name,
					StartTime: md.StartTime,
					StopTime:  md.StopTime,
					StepTime:  md.StepTime,
					Values:    md.Values,
					Wildcard:  len(metrics) == 0,
				})
			}
			result.Metrics = append(result.Metrics, metrics...)
			for _, pattern := range patterns {
				result.Patterns = append(result.Patterns, pattern.Metric)
			}
		}
	}

	return result, nil
}

// GetMetricsTTLSeconds returns metrics lifetime in Redis
func (local *Local) GetMetricsTTLSeconds() int64 {
	return local.dataBase.GetMetricsTTLSeconds()
}

// IsConfigured always returns true. It easy to configure local source =)
func (local *Local) IsConfigured() (bool, error) {
	return true, nil
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
