package local

import (
	"context"
	"math"
	"testing"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/go-graphite/carbonapi/pkg/parser"
	"github.com/go-graphite/protocol/carbonapi_v3_pb"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestLocalFetch(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	const (
		pattern = "test.*"

		from  int64 = 0
		until int64 = 100

		retention      int64 = 10
		retentionFrom  int64 = 0
		retentionUntil int64 = 110
	)

	metrics := []string{"test.test1", "test.test2"}
	expectedValues := [][]float64{
		{math.NaN(), math.NaN(), 0, 1, 2, math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN()},
		{math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(), 3, 4, 5, math.NaN(), math.NaN(), math.NaN()},
	}

	metricValues := map[string][]*moira.MetricValue{
		metrics[0]: {
			{RetentionTimestamp: 20, Timestamp: 23, Value: 0},
			{RetentionTimestamp: 30, Timestamp: 33, Value: 1},
			{RetentionTimestamp: 40, Timestamp: 43, Value: 2},
		},
		metrics[1]: {
			{RetentionTimestamp: 50, Timestamp: 53, Value: 3},
			{RetentionTimestamp: 60, Timestamp: 63, Value: 4},
			{RetentionTimestamp: 70, Timestamp: 73, Value: 5},
		},
	}

	ectx := evalCtx{
		database: database,
	}

	ctx := context.Background()

	Convey("Test Local Fetch", t, func() {
		parsedExpr, _, err := parser.ParseExpr(pattern)
		So(err, ShouldBeNil)

		Convey("Successfully fetched metrics with empty values", func() {
			database.EXPECT().GetPatternMetrics(pattern).Return(metrics, nil).Times(1)
			database.EXPECT().GetMetricRetention(metrics[0]).Return(retention, nil).Times(1)
			database.EXPECT().GetMetricsValues(metrics, retentionFrom, retentionUntil-1).Return(metricValues, nil).Times(1)

			values := make(map[parser.MetricRequest][]*types.MetricData)

			metrics, err := ectx.Fetch(ctx, []parser.Expr{parsedExpr}, from, until, values)
			So(err, ShouldBeNil)

			metricData := metrics[parser.MetricRequest{
				Metric: pattern,
				From:   retentionFrom,
				Until:  retentionUntil,
			}]
			So(cmp.Equal(metricData[0].Values, expectedValues[0], cmpopts.EquateNaNs()), ShouldBeTrue)
			So(cmp.Equal(metricData[1].Values, expectedValues[1], cmpopts.EquateNaNs()), ShouldBeTrue)
		})

		Convey("Successfully fetched metrics with non empty values", func() {
			database.EXPECT().GetPatternMetrics(pattern).Return(metrics, nil).Times(1)
			database.EXPECT().GetMetricRetention(metrics[0]).Return(retention, nil).Times(1)
			database.EXPECT().GetMetricsValues(metrics, retentionFrom, retentionUntil-1).Return(metricValues, nil).Times(1)

			values := make(map[parser.MetricRequest][]*types.MetricData)

			newPattern := "test2.*"

			newValues := []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

			newParsedExpr, _, err := parser.ParseExpr(newPattern)
			So(err, ShouldBeNil)

			newParsedMetricRequest := newParsedExpr.Metrics(0, 0)[0]

			values[newParsedMetricRequest] = []*types.MetricData{
				{
					FetchResponse: carbonapi_v3_pb.FetchResponse{
						Values:    newValues,
						StartTime: retentionFrom,
						StepTime:  retention,
						StopTime:  retentionUntil,
					},
				},
			}

			metrics, err := ectx.Fetch(ctx, []parser.Expr{parsedExpr}, from, until, values)
			So(err, ShouldBeNil)

			metricData := metrics[parser.MetricRequest{
				Metric: pattern,
				From:   retentionFrom,
				Until:  retentionUntil,
			}]
			So(cmp.Equal(metricData[0].Values, expectedValues[0], cmpopts.EquateNaNs()), ShouldBeTrue)
			So(cmp.Equal(metricData[1].Values, expectedValues[1], cmpopts.EquateNaNs()), ShouldBeTrue)

			newParsedMetricRequest.From = retentionFrom
			newParsedMetricRequest.Until = retentionUntil
			metricData = metrics[newParsedMetricRequest]
			So(cmp.Equal(metricData[0].Values, newValues, cmpopts.EquateNaNs()), ShouldBeTrue)
		})
	})
}

func TestLocalEval(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	const (
		from  int64 = 10
		until int64 = 15

		retention      int64 = 1
		retentionFrom  int64 = 10
		retentionUntil int64 = 16
	)

	ectx := evalCtx{
		database: database,
		from:     from,
		until:    until,
	}

	ctx := context.Background()

	Convey("Test Local Eval", t, func() {
		Convey("transformNull(test.*, 0)", func() {
			pattern := "transformNull(test.*, 0)"

			parsedExpr, _, err := parser.ParseExpr(pattern)
			So(err, ShouldBeNil)

			parsedMetricRequest := parsedExpr.Metrics(0, 0)[0]
			parsedMetricRequest.From = from
			parsedMetricRequest.Until = until

			metricValues := []float64{math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN()}

			values := map[parser.MetricRequest][]*types.MetricData{
				parsedMetricRequest: {
					{
						FetchResponse: carbonapi_v3_pb.FetchResponse{
							Values:    metricValues,
							StartTime: retentionFrom,
							StepTime:  retention,
							StopTime:  retentionUntil,
						},
					},
				},
			}

			expectedValues := []float64{0, 0, 0, 0, 0}

			res, err := ectx.Eval(ctx, parsedExpr, from, until, values)
			So(err, ShouldBeNil)

			fetchResp := res[0]
			So(fetchResp.Values, ShouldResemble, expectedValues)
		})

		Convey("movingSum(test.*, '2sec')", func() {
			pattern := "movingSum(test.*, '2sec')"

			var refetchFrom int64 = 8

			parsedExpr, _, err := parser.ParseExpr(pattern)
			So(err, ShouldBeNil)

			parsedMetricRequest := parsedExpr.Metrics(0, 0)[0]
			parsedMetricRequest.From = refetchFrom
			parsedMetricRequest.Until = until

			metricValues := []float64{1, 2, 3, 4, 5}

			values := map[parser.MetricRequest][]*types.MetricData{
				parsedMetricRequest: {
					{
						FetchResponse: carbonapi_v3_pb.FetchResponse{
							Values:    metricValues,
							StartTime: retentionFrom,
							StepTime:  retention,
							StopTime:  retentionUntil,
						},
					},
				},
			}

			expectedValues := []float64{5, 7, 9}

			res, err := ectx.Eval(ctx, parsedExpr, from, until, values)
			So(err, ShouldBeNil)

			fetchResp := res[0]
			So(fetchResp.Values, ShouldResemble, expectedValues)
		})

		Convey("movingSum(test.*, 2)", func() {
			target := "movingSum(test.*, 2)"

			var refetchFrom int64 = 8

			parsedExpr, _, err := parser.ParseExpr(target)
			So(err, ShouldBeNil)

			parsedMetricRequest := parsedExpr.Metrics(0, 0)[0]
			parsedMetricRequest.From = from
			parsedMetricRequest.Until = until

			metricValues := []float64{1, 2, 3, 4, 5}

			values := map[parser.MetricRequest][]*types.MetricData{
				parsedMetricRequest: {
					{
						FetchResponse: carbonapi_v3_pb.FetchResponse{
							Values:    metricValues,
							StartTime: retentionFrom,
							StepTime:  retention,
							StopTime:  retentionUntil,
						},
					},
				},
			}

			patternMetric := "test.test"
			pattern := "test.*"

			valuesMap := map[string][]*moira.MetricValue{
				patternMetric: {
					{RetentionTimestamp: 8, Timestamp: 8, Value: math.NaN()},
					{RetentionTimestamp: 9, Timestamp: 9, Value: math.NaN()},
					{RetentionTimestamp: 10, Timestamp: 10, Value: 1},
					{RetentionTimestamp: 11, Timestamp: 11, Value: 2},
					{RetentionTimestamp: 12, Timestamp: 12, Value: 3},
					{RetentionTimestamp: 13, Timestamp: 13, Value: 4},
					{RetentionTimestamp: 14, Timestamp: 14, Value: 5},
					{RetentionTimestamp: 15, Timestamp: 15, Value: 6},
					{RetentionTimestamp: 16, Timestamp: 16, Value: 7},
				},
			}

			database.EXPECT().GetPatternMetrics(pattern).Return([]string{patternMetric}, nil).Times(1)
			database.EXPECT().GetMetricRetention(patternMetric).Return(retention, nil).Times(1)
			database.EXPECT().GetMetricsValues([]string{patternMetric}, refetchFrom, until).Return(valuesMap, nil).Times(1)

			expectedValues := []float64{1, 3, 5, 7, 9, 11}

			res, err := ectx.Eval(ctx, parsedExpr, from, until, values)
			So(err, ShouldBeNil)

			fetchResp := res[0]
			So(fetchResp.Values, ShouldResemble, expectedValues)
		})

		Convey("applyByNode(test.*, 1, transformNull(%, 0))", func() {
			pattern := "applyByNode(test.*, 1, transformNull(%, 0))"

			parsedExpr, _, err := parser.ParseExpr(pattern)
			So(err, ShouldBeNil)

			parsedMetricRequest := parsedExpr.Metrics(from, until)[0]

			metricValues := []float64{math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN()}

			values := map[parser.MetricRequest][]*types.MetricData{
				parsedMetricRequest: {
					{
						FetchResponse: carbonapi_v3_pb.FetchResponse{
							Values:    metricValues,
							StartTime: retentionFrom,
							StepTime:  retention,
							StopTime:  retentionUntil,
						},
					},
				},
			}

			expectedValues := []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

			res, err := ectx.Eval(ctx, parsedExpr, from, until, values)
			So(err, ShouldBeNil)

			fetchResp := res[0]
			So(fetchResp.Values, ShouldResemble, expectedValues)
		})
	})
}

func TestLocalFetchAndEval(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	const (
		pattern       = "test.*"
		matchedMetric = "test.test1"

		from  int64 = 10
		until int64 = 15

		retention      int64 = 1
		retentionFrom  int64 = 8
		retentionUntil int64 = 15
	)

	metricValues := map[string][]*moira.MetricValue{
		matchedMetric: {
			{RetentionTimestamp: 8, Timestamp: 8, Value: 1},
			{RetentionTimestamp: 9, Timestamp: 9, Value: 2},
			{RetentionTimestamp: 10, Timestamp: 10, Value: 1},
			{RetentionTimestamp: 11, Timestamp: 11, Value: 2},
			{RetentionTimestamp: 12, Timestamp: 12, Value: 3},
			{RetentionTimestamp: 13, Timestamp: 13, Value: 4},
			{RetentionTimestamp: 14, Timestamp: 14, Value: 5},
			{RetentionTimestamp: 15, Timestamp: 15, Value: 6},
		},
	}

	Convey("Test Local FetchAndEval", t, func() {
		ectx := evalCtx{
			database: database,
			from:     from,
			until:    until,
		}

		Convey("movingAverage(test.*, '2sec', 0)", func() {
			target := "movingAverage(test.*, '2sec', 0)"

			fetchResult := &FetchResult{
				MetricsData: make([]metricSource.MetricData, 0),
				Patterns:    make([]string, 0),
				Metrics:     make([]string, 0),
			}

			database.EXPECT().GetPatternMetrics(pattern).Return([]string{matchedMetric}, nil)
			database.EXPECT().GetMetricRetention(matchedMetric).Return(retention, nil)
			database.EXPECT().GetMetricsValues([]string{matchedMetric}, retentionFrom, until).Return(metricValues, nil)

			expectedResult := &FetchResult{
				MetricsData: []metricSource.MetricData{
					{
						Name:      "movingAverage(test.test1,'2sec')",
						StartTime: 10,
						StopTime:  16,
						StepTime:  1,
						Values:    []float64{1.5, 1.5, 2.5, 3.5, 4.5, 5.5},
					},
				},
				Patterns: []string{pattern},
				Metrics:  []string{matchedMetric},
			}

			err := ectx.fetchAndEval(target, fetchResult)
			So(err, ShouldBeNil)
			So(fetchResult, ShouldResemble, expectedResult)
		})

		Convey("movingSum(test.*, 2)", func() {
			target := "movingSum(test.*, 2)"

			fetchResult := &FetchResult{
				MetricsData: make([]metricSource.MetricData, 0),
				Patterns:    make([]string, 0),
				Metrics:     make([]string, 0),
			}

			database.EXPECT().GetPatternMetrics(pattern).Return([]string{matchedMetric}, nil).Times(2)
			database.EXPECT().GetMetricRetention(matchedMetric).Return(retention, nil).Times(2)
			database.EXPECT().GetMetricsValues([]string{matchedMetric}, from, until).Return(metricValues, nil).Times(1)
			database.EXPECT().GetMetricsValues([]string{matchedMetric}, retentionFrom, retentionUntil).Return(metricValues, nil).Times(1)

			expectedResult := &FetchResult{
				MetricsData: []metricSource.MetricData{
					{
						Name:      "movingSum(test.test1,2)",
						StartTime: 10,
						StopTime:  16,
						StepTime:  1,
						Values:    []float64{3, 3, 5, 7, 9, 11},
					},
				},
				Patterns: []string{pattern},
				Metrics:  []string{matchedMetric},
			}

			err := ectx.fetchAndEval(target, fetchResult)
			So(err, ShouldBeNil)
			So(fetchResult, ShouldResemble, expectedResult)
		})
	})
}
