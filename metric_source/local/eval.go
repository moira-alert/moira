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
	from     int64
	until    int64
	database moira.Database
}

// Fetch fetch metrics (for compatibility with carbonapi Evaluator interface).
func (ectx *evalCtx) Fetch(ctx context.Context, exprs []parser.Expr, from, until int64, values map[parser.MetricRequest][]*types.MetricData) (map[parser.MetricRequest][]*types.MetricData, error) {
	fetchedMetrics := fetchedMetrics{
		metrics:    make([]string, 0),
		metricsMap: values,
	}

	for _, exp := range exprs {
		if err := ectx.getMetricsData(exp.Metrics(0, 0), &fetchedMetrics); err != nil {
			return nil, err
		}
	}

	fetchedMetrics.metricsMap = helper.ScaleValuesToCommonStep(fetchedMetrics.metricsMap)

	return fetchedMetrics.metricsMap, nil
}

// Eval evaluates expressions (for compatibility with carbonapi Evaluator interface).
func (ectx *evalCtx) Eval(ctx context.Context, exp parser.Expr, from, until int64, values map[parser.MetricRequest][]*types.MetricData) (results []*types.MetricData, err error) {
	rewritten, targets, err := expr.RewriteExpr(ctx, ectx, exp, from, until, values)
	if err != nil {
		return nil, err
	}

	if rewritten {
		for _, target := range targets {
			exp, _, err = parser.ParseExpr(target)
			if err != nil {
				return nil, err
			}

			targetValues, err := ectx.Fetch(ctx, []parser.Expr{exp}, from, until, values)
			if err != nil {
				return nil, err
			}

			result, err := ectx.Eval(ctx, exp, from, until, targetValues)
			if err != nil {
				return nil, err
			}

			results = append(results, result...)
		}

		return results, nil
	}

	return expr.EvalExpr(ctx, ectx, exp, from, until, values)
}

func (ectx *evalCtx) fetchAndEval(target string, result *FetchResult) error {
	exp, err := ectx.parse(target)
	if err != nil {
		return err
	}

	fetchedMetrics := fetchedMetrics{
		metrics:    make([]string, 0),
		metricsMap: make(map[parser.MetricRequest][]*types.MetricData),
	}

	if err := ectx.getMetricsData(exp.Metrics(0, 0), &fetchedMetrics); err != nil {
		return err
	}

	commonStep := fetchedMetrics.calculateCommonStep()
	ectx.scaleToCommonStep(commonStep, &fetchedMetrics)

	rewritten, newTargets, err := ectx.rewriteExpr(exp, &fetchedMetrics)
	if err != nil {
		return err
	}

	if rewritten {
		for _, newTarget := range newTargets {
			err = ectx.fetchAndEvalNoRewrite(newTarget, result)
			if err != nil {
				return err
			}
		}

		return nil
	}

	metricsData, err := ectx.eval(target, exp, &fetchedMetrics)
	if err != nil {
		return err
	}

	ectx.writeResult(exp, &fetchedMetrics, metricsData, result)

	return nil
}

func (ectx *evalCtx) fetchAndEvalNoRewrite(target string, result *FetchResult) error {
	exp, err := ectx.parse(target)
	if err != nil {
		return err
	}

	fetchedMetrics := fetchedMetrics{
		metrics:    make([]string, 0),
		metricsMap: make(map[parser.MetricRequest][]*types.MetricData),
	}

	if err := ectx.getMetricsData(exp.Metrics(0, 0), &fetchedMetrics); err != nil {
		return err
	}

	commonStep := fetchedMetrics.calculateCommonStep()
	ectx.scaleToCommonStep(commonStep, &fetchedMetrics)

	metricsData, err := ectx.eval(target, exp, &fetchedMetrics)
	if err != nil {
		return err
	}

	ectx.writeResult(exp, &fetchedMetrics, metricsData, result)

	return nil
}

func (ectx *evalCtx) parse(target string) (parser.Expr, error) {
	parsedExpr, _, err := parser.ParseExpr(target)
	if err != nil {
		return nil, ErrParseExpr{
			internalError: err,
			target:        target,
		}
	}

	return parsedExpr, nil
}

func (ectx *evalCtx) getMetricsData(metricRequests []parser.MetricRequest, result *fetchedMetrics) error {
	fetchData := fetchData{ectx.database}

	for _, mr := range metricRequests {
		if _, ok := result.metricsMap[mr]; ok {
			continue
		}

		from := mr.From + ectx.from
		until := mr.Until + ectx.until

		metricNames, err := fetchData.fetchMetricNames(mr.Metric)
		if err != nil {
			return err
		}

		timer := NewTimerRoundingTimestamps(from, until, metricNames.retention)

		metricsData, err := fetchData.fetchMetricValues(mr.Metric, metricNames, timer)
		if err != nil {
			return err
		}

		result.metricsMap[mr] = metricsData
		result.metrics = append(result.metrics, metricNames.metrics...)
	}

	return nil
}

func (ectx *evalCtx) scaleToCommonStep(retention int64, fetchedMetrics *fetchedMetrics) {
	from, until := RoundTimestamps(ectx.from, ectx.until, retention)
	ectx.from, ectx.until = from, until

	metricMap := make(map[parser.MetricRequest][]*types.MetricData)
	for metricRequest, metricData := range fetchedMetrics.metricsMap {
		metricRequest.From += from
		metricRequest.Until += until

		metricData = helper.ScaleToCommonStep(metricData, retention)
		metricMap[metricRequest] = metricData
	}

	fetchedMetrics.metricsMap = metricMap
}

func (ectx *evalCtx) rewriteExpr(parsedExpr parser.Expr, metrics *fetchedMetrics) (bool, []string, error) {
	rewritten, newTargets, err := expr.RewriteExpr(
		context.Background(),
		ectx,
		parsedExpr,
		ectx.from,
		ectx.until,
		metrics.metricsMap,
	)

	if err != nil && !errors.Is(err, parser.ErrMissingTimeseries) {
		return false, nil, fmt.Errorf("failed to RewriteExpr: %w", err)
	}

	return rewritten, newTargets, nil
}

func (ectx *evalCtx) eval(target string, parsedExpr parser.Expr, metrics *fetchedMetrics) (result []*types.MetricData, err error) {
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

	result, err = expr.EvalExpr(context.Background(), ectx, parsedExpr, ectx.from, ectx.until, metrics.metricsMap)
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

func (ectx *evalCtx) writeResult(exp parser.Expr, metrics *fetchedMetrics, metricsData []*types.MetricData, result *FetchResult) {
	for _, metricData := range metricsData {
		md := newMetricDataFromGraphite(metricData, metrics.hasWildcard())
		result.MetricsData = append(result.MetricsData, md)
	}

	result.Metrics = append(result.Metrics, metrics.metrics...)
	for _, mr := range exp.Metrics(0, 0) {
		result.Patterns = append(result.Patterns, mr.Metric)
	}
}

func newMetricDataFromGraphite(md *types.MetricData, wildcard bool) metricSource.MetricData {
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

func (m *fetchedMetrics) hasWildcard() bool {
	return len(m.metrics) == 0
}

func (m *fetchedMetrics) calculateCommonStep() int64 {
	commonStep := int64(1)
	for _, metricsData := range m.metricsMap {
		for _, metricData := range metricsData {
			commonStep = helper.LCM(commonStep, metricData.StepTime)
		}
	}

	return commonStep
}
