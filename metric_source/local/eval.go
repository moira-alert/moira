package local

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/go-graphite/carbonapi/expr"
	"github.com/go-graphite/carbonapi/expr/helper"
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/go-graphite/carbonapi/pkg/parser"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
)

type evalCtx struct {
	from  int64
	until int64
}

func (ctx *evalCtx) FetchAndEval(database moira.Database, target string, result *FetchResult) error {
	expr, err := ctx.Parse(target)
	if err != nil {
		return err
	}

	fetchedMetrics, err := ctx.GetMetricsData(database, expr)
	if err != nil {
		return err
	}

	commonStep := fetchedMetrics.CalculateCommonStep()
	ctx.ScaleToCommonStep(commonStep, fetchedMetrics)

	rewritten, newTargets, err := ctx.RewriteExpr(expr, fetchedMetrics)
	if err != nil {
		return err
	}

	if rewritten {
		for _, newTarget := range newTargets {
			err = ctx.FetchAndEval(database, newTarget, result)
			if err != nil {
				return err
			}
		}
		return nil
	}

	metricsData, err := ctx.Eval(target, expr, fetchedMetrics)
	if err != nil {
		return err
	}

	for _, metricData := range metricsData {
		md := MetricDataFromGraphit(metricData, fetchedMetrics.HasWildcard())
		result.MetricsData = append(result.MetricsData, md)
	}

	result.Metrics = append(result.Metrics, fetchedMetrics.metrics...)
	for _, mr := range expr.Metrics() {
		result.Patterns = append(result.Patterns, mr.Metric)
	}

	return nil
}

func (ctx *evalCtx) Parse(target string) (parser.Expr, error) {
	parsedExpr, _, err := parser.ParseExpr(target)
	if err != nil {
		return nil, ErrParseExpr{
			internalError: err,
			target:        target,
		}
	}
	return parsedExpr, nil
}

func (ctx *evalCtx) GetMetricsData(database moira.Database, parsedExpr parser.Expr) (*fetchedMetrics, error) {
	metricRequests := parsedExpr.Metrics()

	metrics := make([]string, 0)
	metricsMap := make(map[parser.MetricRequest][]*types.MetricData)

	fetchData := fetchData{database}

	for _, mr := range metricRequests {
		from := mr.From + ctx.from
		until := mr.Until + ctx.until

		metricNames, err := fetchData.fetchMetricNames(mr.Metric)
		if err != nil {
			return nil, err
		}

		timer := NewTimerRoundingTimestamps(from, until, metricNames.retention)

		metricsData, err := fetchData.fetchMetricValues(mr.Metric, metricNames, timer)
		if err != nil {
			return nil, err
		}

		metricsMap[mr] = metricsData
		metrics = append(metrics, metricNames.metrics...)
	}
	return &fetchedMetrics{metricsMap, metrics}, nil
}

func (ctx *evalCtx) ScaleToCommonStep(retention int64, fetchedMetrics *fetchedMetrics) {
	from, until := RoundTimestamps(ctx.from, ctx.until, retention)
	ctx.from, ctx.until = from, until

	metricMap := make(map[parser.MetricRequest][]*types.MetricData)
	for metricRequest, metricData := range fetchedMetrics.metricsMap {
		metricRequest.From += from
		metricRequest.Until += until

		metricData = helper.ScaleToCommonStep(metricData, retention)
		metricMap[metricRequest] = metricData
	}

	fetchedMetrics.metricsMap = metricMap
}

func (ctx *evalCtx) RewriteExpr(parsedExpr parser.Expr, metrics *fetchedMetrics) (bool, []string, error) {
	rewritten, newTargets, err := expr.RewriteExpr(
		context.Background(),
		parsedExpr,
		ctx.from,
		ctx.until,
		metrics.metricsMap,
	)

	if err != nil && err != parser.ErrMissingTimeseries {
		return false, nil, fmt.Errorf("failed RewriteExpr: %s", err.Error())
	}
	return rewritten, newTargets, nil
}

func (ctx *evalCtx) Eval(target string, parsedExpr parser.Expr, metrics *fetchedMetrics) (result []*types.MetricData, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = ErrEvaluateTargetFailedWithPanic{
				target:         target,
				recoverMessage: r,
				stackRecord:    debug.Stack(),
			}
		}
	}()

	result, err = expr.EvalExpr(context.Background(), parsedExpr, ctx.from, ctx.until, metrics.metricsMap)
	if err != nil {
		if errors.Is(err, parser.ErrMissingTimeseries) {
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
}

func MetricDataFromGraphit(md *types.MetricData, wildcard bool) metricSource.MetricData {
	return metricSource.MetricData{
		Name:      md.Name,
		StartTime: md.StartTime,
		StopTime:  md.StopTime,
		StepTime:  md.StepTime,
		Values:    md.Values,
		Wildcard:  wildcard,
	}
}

type fetchedMetrics struct {
	metricsMap map[parser.MetricRequest][]*types.MetricData
	metrics    []string
}

func (m *fetchedMetrics) HasWildcard() bool {
	return len(m.metrics) == 0
}

func (m *fetchedMetrics) CalculateCommonStep() int64 {
	commonStep := int64(1)
	for _, metricsData := range m.metricsMap {
		for _, metricData := range metricsData {
			commonStep = helper.LCM(commonStep, metricData.StepTime)
		}
	}
	return commonStep
}
