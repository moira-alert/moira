package local

import (
	"context"
	"errors"
	"runtime/debug"

	"github.com/ansel1/merry"
	"github.com/go-graphite/carbonapi/expr"
	"github.com/go-graphite/carbonapi/expr/helper"
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/go-graphite/carbonapi/pkg/parser"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
)

type evaluator struct {
	database moira.Database
	metrics  []string
}

func (eval *evaluator) fetchAndEval(target string, from, until int64, result *FetchResult) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrEvaluateTargetFailedWithPanic{
				target:         target,
				recoverMessage: r,
				stackRecord:    debug.Stack(),
			}
		}
	}()

	exp, err := eval.parse(target)
	if err != nil {
		return err
	}

	values := make(map[parser.MetricRequest][]*types.MetricData)

	fetchedMetrics, err := expr.FetchAndEvalExp(context.Background(), eval, exp, from, until, values)
	if err != nil {
		return merry.Unwrap(err)
	}

	eval.writeResult(exp, fetchedMetrics, result)

	return nil
}

// Fetch is an implementation of Evaluator interface from carbonapi.
// It returns a map the metrics requested in the current invocation, scaled to a common step.
func (eval *evaluator) Fetch(
	ctx context.Context,
	exprs []parser.Expr,
	from, until int64,
	values map[parser.MetricRequest][]*types.MetricData,
) (map[parser.MetricRequest][]*types.MetricData, error) {
	fetch := newFetchCtx(0, 0)

	for _, exp := range exprs {
		ms := exp.Metrics(from, until)
		if err := fetch.getMetricsData(eval.database, ms); err != nil {
			return nil, err
		}
	}

	fetch.scaleToCommonStep()

	eval.metrics = append(eval.metrics, fetch.fetchedMetrics.metrics...)

	return fetch.fetchedMetrics.metricsMap, nil
}

// Eval is an implementation of Evaluator interface from carbonapi.
// It uses the raw data within the values map being passed into it to in order to evaluate the input expression.
func (eval *evaluator) Eval(
	ctx context.Context,
	exp parser.Expr,
	from, until int64,
	values map[parser.MetricRequest][]*types.MetricData,
) (results []*types.MetricData, err error) {
	rewritten, newTargets, err := expr.RewriteExpr(ctx, eval, exp, from, until, values)
	if err != nil {
		return nil, err
	}

	if rewritten {
		return eval.evalRewritten(ctx, newTargets, from, until, values)
	}

	results, err = expr.EvalExpr(ctx, eval, exp, from, until, values)
	if err != nil {
		if errors.Is(err, parser.ErrMissingTimeseries) {
			err = nil
		} else if isErrUnknownFunction(err) {
			err = ErrorUnknownFunction(err)
		} else {
			err = ErrEvalExpr{
				target:        exp.ToString(),
				internalError: err,
			}
		}
	}

	return results, err
}

func (eval *evaluator) evalRewritten(
	ctx context.Context,
	newTargets []string,
	from, until int64,
	values map[parser.MetricRequest][]*types.MetricData,
) (results []*types.MetricData, err error) {
	for _, target := range newTargets {
		exp, _, err := parser.ParseExpr(target)
		if err != nil {
			return nil, err
		}

		var targetValues map[parser.MetricRequest][]*types.MetricData
		targetValues, err = eval.Fetch(ctx, []parser.Expr{exp}, from, until, values)
		if err != nil {
			return nil, err
		}

		result, err := eval.Eval(ctx, exp, from, until, targetValues)
		if err != nil {
			return nil, err
		}

		results = append(results, result...)
	}

	return results, nil
}

func (eval *evaluator) writeResult(exp parser.Expr, metricsData []*types.MetricData, result *FetchResult) {
	result.Metrics = append(result.Metrics, eval.metrics...)
	for _, mr := range exp.Metrics(0, 0) {
		result.Patterns = append(result.Patterns, mr.Metric)
	}

	for _, metricData := range metricsData {
		md := newMetricDataFromGraphite(metricData, len(result.Metrics) != len(result.Patterns))
		result.MetricsData = append(result.MetricsData, md)
	}
}

func (eval *evaluator) parse(target string) (parser.Expr, error) {
	parsedExpr, _, err := parser.ParseExpr(target)
	if err != nil {
		return nil, ErrParseExpr{
			internalError: err,
			target:        target,
		}
	}
	return parsedExpr, nil
}

type fetchCtx struct {
	from           int64
	until          int64
	fetchedMetrics *fetchedMetrics
}

func newFetchCtx(from, until int64) *fetchCtx {
	return &fetchCtx{
		from,
		until,
		&fetchedMetrics{
			metricsMap: make(map[parser.MetricRequest][]*types.MetricData),
			metrics:    make([]string, 0),
		},
	}
}

func (ctx *fetchCtx) getMetricsData(database moira.Database, metricRequests []parser.MetricRequest) error {
	fetchData := fetchData{database}

	for _, mr := range metricRequests {
		// Other fields are used in carbon for database side consolidations
		request := parser.MetricRequest{
			Metric: mr.Metric,
			From:   mr.From,
			Until:  mr.Until,
		}

		from := mr.From + ctx.from
		until := mr.Until + ctx.until

		metricNames, err := fetchData.fetchMetricNames(mr.Metric)
		if err != nil {
			return err
		}

		timer := newTimerRoundingTimestamps(from, until, metricNames.retention)

		metricsData, err := fetchData.fetchMetricValues(mr.Metric, metricNames, timer)
		if err != nil {
			return err
		}

		ctx.fetchedMetrics.metricsMap[request] = metricsData
		ctx.fetchedMetrics.metrics = append(ctx.fetchedMetrics.metrics, metricNames.metrics...)
	}
	return nil
}

func (ctx *fetchCtx) scaleToCommonStep() {
	retention := ctx.fetchedMetrics.calculateCommonStep()

	metricMap := make(map[parser.MetricRequest][]*types.MetricData)
	for metricRequest, metricData := range ctx.fetchedMetrics.metricsMap {
		metricRequest.From += ctx.from
		metricRequest.Until += ctx.until

		metricData = helper.ScaleToCommonStep(metricData, retention)
		metricMap[metricRequest] = metricData
	}

	ctx.fetchedMetrics.metricsMap = metricMap
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

func (m *fetchedMetrics) calculateCommonStep() int64 {
	commonStep := int64(1)
	for _, metricsData := range m.metricsMap {
		for _, metricData := range metricsData {
			commonStep = helper.LCM(commonStep, metricData.StepTime)
		}
	}
	return commonStep
}
