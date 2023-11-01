package local

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/go-graphite/carbonapi/expr/functions"
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/go-graphite/carbonapi/pkg/parser"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	functions.New(make(map[string]string))
}

const (
	pattern1 = "super.puper.pattern"
	pattern2 = "super.duper.pattern"
	metric1  = "super.puper.metric"
	metric2  = "super.duper.metric"
)

func TestLocalSourceFetchErrors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	localSource := Create(database)
	defer mockCtrl.Finish()

	dataList := map[string][]*moira.MetricValue{
		metric1: {
			{RetentionTimestamp: 20, Timestamp: 23, Value: 0},
			{RetentionTimestamp: 30, Timestamp: 33, Value: 1},
			{RetentionTimestamp: 40, Timestamp: 43, Value: 2},
			{RetentionTimestamp: 50, Timestamp: 53, Value: 3},
			{RetentionTimestamp: 60, Timestamp: 63, Value: 4},
		},
	}

	var from int64 = 17
	var until int64 = 67
	var retentionFrom int64 = 20
	var retentionUntil int64 = 70
	var retention int64 = 10
	var metricsTTL int64 = 3600
	metricErr := fmt.Errorf("Ooops, metric error")

	Convey("Error while ParseExpr", t, func() {
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch("", from, until, true)

		So(err, ShouldResemble, ErrParseExpr{target: "", internalError: parser.ErrMissingExpr})
		So(err.Error(), ShouldResemble, "failed to parse target '': missing expression")
		So(result, ShouldBeNil)
	})

	Convey("Error in fetch data", t, func() {
		database.EXPECT().GetPatternMetrics(pattern1).Return([]string{metric1}, nil)
		database.EXPECT().GetMetricRetention(metric1).Return(retention, nil)
		database.EXPECT().GetMetricsValues([]string{metric1}, retentionFrom, retentionUntil-1).Return(nil, metricErr)
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch(pattern1, from, until, true)

		So(err, ShouldResemble, metricErr)
		So(result, ShouldBeNil)
	})

	Convey("Error evaluate target", t, func() {
		database.EXPECT().GetPatternMetrics(pattern1).Return([]string{metric1}, nil)
		database.EXPECT().GetMetricRetention(metric1).Return(retention, nil)
		database.EXPECT().GetMetricsValues([]string{metric1}, retentionFrom, retentionUntil-1).Return(dataList, nil)
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch("aliasByNoe(super.puper.pattern, 2)", from, until, true)

		So(err.Error(), ShouldResemble, "Unknown graphite function: \"aliasByNoe\"")
		So(result, ShouldBeNil)
	})

	Convey("Panic while evaluate target", t, func() {
		database.EXPECT().GetPatternMetrics(pattern1).Return([]string{metric1}, nil)
		database.EXPECT().GetMetricRetention(metric1).Return(retention, nil)
		database.EXPECT().GetMetricsValues([]string{metric1}, retentionFrom, retentionUntil-1).Return(dataList, nil)
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch("movingAverage(super.puper.pattern, -1)", from, until, true)
		expectedErrSubstring := strings.Split(ErrEvaluateTargetFailedWithPanic{target: "movingAverage(super.puper.pattern, -1)"}.Error(), ":")[0]

		So(err.Error(), ShouldStartWith, expectedErrSubstring)
		So(result, ShouldBeNil)
	})
}

func TestLocalSourceFetchNoMetrics(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	localSource := Create(database)
	defer mockCtrl.Finish()

	pattern := pattern1
	pattern2 := pattern2

	var metricsTTL int64 = 3600

	Convey("Single pattern, from 17 until 17", t, func() {
		database.EXPECT().GetPatternMetrics(pattern).Return([]string{}, nil)
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch("aliasByNode(super.puper.pattern, 2)", 17, 17, false)

		So(err, ShouldBeNil)
		So(result, shouldEqualIfNaNsEqual, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "pattern",
				StartTime: 60,
				StopTime:  60,
				StepTime:  60,
				Values:    []float64{},
				Wildcard:  true,
			}},
			Metrics:  []string{},
			Patterns: []string{pattern},
		})
	})

	Convey("Single pattern, from 17 until 67", t, func() {
		database.EXPECT().GetPatternMetrics(pattern).Return([]string{}, nil)
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch("aliasByNode(super.puper.pattern, 2)", 17, 67, true)

		So(err, ShouldBeNil)
		So(result, shouldEqualIfNaNsEqual, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "pattern",
				StartTime: 60,
				StopTime:  120,
				StepTime:  60, Values: []float64{math.NaN()},
				Wildcard: true,
			}},
			Metrics:  []string{},
			Patterns: []string{pattern},
		})
	})

	Convey("Single pattern, from 7 until 57", t, func() {
		database.EXPECT().GetPatternMetrics(pattern).Return([]string{}, nil)
		database.EXPECT().GetPatternMetrics(pattern2).Return([]string{}, nil)
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch("aliasByNode(super.puper.pattern, 2)", 7, 57, true)

		So(err, ShouldBeNil)
		So(result, shouldEqualIfNaNsEqual, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "pattern",
				StartTime: 60,
				StopTime:  60,
				StepTime:  60, Values: []float64{},
				Wildcard: true,
			}},
			Metrics:  []string{},
			Patterns: []string{pattern},
		})
	})

	Convey("Two patterns, from 17 until 67", t, func() {
		database.EXPECT().GetPatternMetrics(pattern).Return([]string{}, nil)
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch("alias(sum(super.puper.pattern, super.duper.pattern), 'pattern')", 17, 67, true)

		So(err, ShouldBeNil)
		So(result, shouldEqualIfNaNsEqual, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "pattern",
				StartTime: 60,
				StopTime:  120,
				StepTime:  60, Values: []float64{math.NaN()},
				Wildcard: true,
			}},
			Metrics:  []string{},
			Patterns: []string{pattern, pattern2},
		})
	})
}

func TestLocalSourceFetchMultipleMetrics(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	localSource := Create(database)
	defer mockCtrl.Finish()

	var from int64 = 17
	var until int64 = 67
	var retentionFrom int64 = 20
	var retentionUntil int64 = 70
	var retention int64 = 10
	var metricsTTL int64 = 3600

	Convey("Test success evaluate multiple metrics with pow function", t, func() {
		metrics := []string{
			"apps.server1.process.cpu.usage",
			"apps.server2.process.cpu.usage",
			"apps.server3.process.cpu.usage",
		}

		multipleDataList := make(map[string][]*moira.MetricValue)
		multipleDataList["apps.server1.process.cpu.usage"] = []*moira.MetricValue{
			{RetentionTimestamp: 20, Timestamp: 23, Value: 0.5},
			{RetentionTimestamp: 30, Timestamp: 33, Value: 0.4},
			{RetentionTimestamp: 40, Timestamp: 43, Value: 0.5},
			{RetentionTimestamp: 50, Timestamp: 53, Value: 0.5},
			{RetentionTimestamp: 60, Timestamp: 63, Value: 0.5},
		}
		multipleDataList["apps.server2.process.cpu.usage"] = []*moira.MetricValue{
			{RetentionTimestamp: 20, Timestamp: 23, Value: math.NaN()},
			{RetentionTimestamp: 30, Timestamp: 33, Value: math.NaN()},
			{RetentionTimestamp: 40, Timestamp: 43, Value: math.NaN()},
			{RetentionTimestamp: 50, Timestamp: 53, Value: math.NaN()},
			{RetentionTimestamp: 60, Timestamp: 63, Value: math.NaN()},
		}
		multipleDataList["apps.server3.process.cpu.usage"] = []*moira.MetricValue{
			{RetentionTimestamp: 20, Timestamp: 23, Value: 0.5},
			{RetentionTimestamp: 30, Timestamp: 33, Value: 0.5},
			{RetentionTimestamp: 40, Timestamp: 43, Value: 0.5},
			{RetentionTimestamp: 50, Timestamp: 53, Value: 0.4},
			{RetentionTimestamp: 60, Timestamp: 63, Value: 0.5},
		}

		database.EXPECT().GetPatternMetrics("apps.*.process.cpu.usage").Return(metrics, nil)
		database.EXPECT().GetMetricRetention(metrics[0]).Return(retention, nil)
		database.EXPECT().GetMetricsValues(metrics, retentionFrom, retentionUntil-1).Return(multipleDataList, nil)
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch("alias(sumSeries(pow(apps.*.process.cpu.usage, 0)), 'alive replicas')", from, until, true)

		So(err, ShouldBeNil)
		So(result, shouldEqualIfNaNsEqual, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "alive replicas",
				StartTime: retentionFrom,
				StopTime:  retentionUntil,
				StepTime:  retention,
				Values:    []float64{2, 2, 2, 2, 2},
			},
			},
			Metrics:  metrics,
			Patterns: []string{"apps.*.process.cpu.usage"},
		})
	})
}

func TestLocalSourceFetch(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	localSource := Create(database)
	defer mockCtrl.Finish()

	pattern := pattern1
	metric := metric1
	dataList := map[string][]*moira.MetricValue{
		metric: {
			{RetentionTimestamp: 20, Timestamp: 23, Value: 0},
			{RetentionTimestamp: 30, Timestamp: 33, Value: 1},
			{RetentionTimestamp: 40, Timestamp: 43, Value: 2},
			{RetentionTimestamp: 50, Timestamp: 53, Value: 3},
			{RetentionTimestamp: 60, Timestamp: 63, Value: 4},
		},
	}

	var from int64 = 17
	var until int64 = 67
	var retentionFrom int64 = 20
	var retentionUntil int64 = 70
	var retention int64 = 10
	var metricsTTL int64 = 3600

	Convey("Test success evaluate", t, func() {
		database.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		database.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		database.EXPECT().GetMetricsValues([]string{metric}, retentionFrom, retentionUntil-1).Return(dataList, nil)
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch("aliasByNode(super.puper.pattern, 2)", from, until, true)
		So(err, ShouldBeNil)
		So(result, shouldEqualIfNaNsEqual, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "metric",
				StartTime: retentionFrom,
				StopTime:  retentionUntil,
				StepTime:  retention, Values: []float64{0, 1, 2, 3, 4},
			},
			},
			Metrics:  []string{metric},
			Patterns: []string{pattern},
		})
	})

	Convey("Test enormous fetch interval", t, func() {
		var fromPast int64 = 0
		var toFuture int64 = 1e15
		var ttl = 2*retention - 1

		var distantFutureDataList = map[string][]*moira.MetricValue{
			metric: {
				{RetentionTimestamp: toFuture, Timestamp: toFuture, Value: 0},
				{RetentionTimestamp: toFuture - retention, Timestamp: toFuture - retention, Value: 0},
			},
		}

		database.EXPECT().GetPatternMetrics(pattern1).Return([]string{metric}, nil)
		database.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		database.EXPECT().GetMetricsValues([]string{metric}, toFuture-retention, toFuture+retention-1).Return(distantFutureDataList, nil)
		database.EXPECT().GetMetricsTTLSeconds().Return(ttl)

		result, err := localSource.Fetch("aliasByNode(super.puper.pattern, 2)", fromPast, toFuture, true)

		So(err, ShouldBeNil)
		So(result, shouldEqualIfNaNsEqual, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "metric",
				StartTime: toFuture - retention,
				StopTime:  toFuture + retention,
				StepTime:  retention,
				Values:    []float64{0, 0},
			},
			},
			Metrics:  []string{metric},
			Patterns: []string{pattern1},
		})
	})

	Convey("Test success evaluate pipe target", t, func() {
		database.EXPECT().GetPatternMetrics(pattern1).Return([]string{metric}, nil)
		database.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		database.EXPECT().GetMetricsValues([]string{metric}, retentionFrom, retentionUntil-1).Return(dataList, nil)
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch("super.puper.pattern | scale(100) | aliasByNode(2)", from, until, true)

		So(err, ShouldBeNil)
		So(result, ShouldResemble, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "metric",
				StartTime: retentionFrom,
				StopTime:  retentionUntil,
				StepTime:  retention,
				Values:    []float64{0, 100, 200, 300, 400},
			}},
			Metrics:  []string{metric},
			Patterns: []string{pattern1},
		})
	})

	Convey("Test success evaluate target with aliasByTags('name')", t, func() {
		database.EXPECT().GetPatternMetrics(pattern1).Return([]string{metric}, nil)
		database.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		database.EXPECT().GetMetricsValues([]string{metric}, retentionFrom, retentionUntil-1).Return(dataList, nil)
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch("super.puper.pattern | aliasByTags('name')", from, until, true)

		So(err, ShouldBeNil)
		So(result, ShouldResemble, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "super.puper.metric",
				StartTime: retentionFrom,
				StopTime:  retentionUntil,
				StepTime:  retention,
				Values:    []float64{0, 1, 2, 3, 4},
			}},
			Metrics:  []string{metric},
			Patterns: []string{pattern1},
		})
	})

	Convey("Test success evaluate target with currentAbove(3.99)", t, func() {
		database.EXPECT().GetPatternMetrics(pattern1).Return([]string{metric}, nil)
		database.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		database.EXPECT().GetMetricsValues([]string{metric}, retentionFrom, retentionUntil-1).Return(dataList, nil)
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch("super.puper.pattern | currentAbove(3.99)", from, until, true)

		So(err, ShouldBeNil)
		So(result, ShouldResemble, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "super.puper.metric",
				StartTime: retentionFrom,
				StopTime:  retentionUntil,
				StepTime:  retention,
				Values:    []float64{0, 1, 2, 3, 4},
			}},
			Metrics:  []string{metric},
			Patterns: []string{pattern1},
		})
	})

	Convey("Test success evaluate target with currentAbove(4)", t, func() {
		database.EXPECT().GetPatternMetrics(pattern1).Return([]string{metric}, nil)
		database.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		database.EXPECT().GetMetricsValues([]string{metric}, retentionFrom, retentionUntil-1).Return(dataList, nil)
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch("super.puper.pattern | currentAbove(4)", from, until, true)

		So(err, ShouldBeNil)
		So(result, ShouldResemble, &FetchResult{
			MetricsData: []metricSource.MetricData{},
			Metrics:     []string{metric},
			Patterns:    []string{pattern1},
		})
	})
}

func TestLocalSourceFetchNoRealTimeAlerting(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	localSource := Create(database)
	defer mockCtrl.Finish()

	pattern := pattern1
	metric := metric1
	dataList := map[string][]*moira.MetricValue{
		metric: {
			{RetentionTimestamp: 20, Timestamp: 23, Value: 0},
			{RetentionTimestamp: 30, Timestamp: 33, Value: 1},
			{RetentionTimestamp: 40, Timestamp: 43, Value: 2},
			{RetentionTimestamp: 50, Timestamp: 53, Value: 3},
			{RetentionTimestamp: 60, Timestamp: 63, Value: 4},
		},
	}

	var from int64 = 17
	var until int64 = 67
	var retentionFrom int64 = 20
	var retentionUntil int64 = 70
	var retention int64 = 10
	var metricsTTL int64 = 3600

	Convey("Test success evaluate without realtime alerting", t, func() {
		database.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		database.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		database.EXPECT().GetMetricsValues([]string{metric}, retentionFrom, retentionUntil-1).Return(dataList, nil)
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch("aliasByNode(super.puper.pattern, 2)", from, until, false)
		So(err, ShouldBeNil)
		So(result, shouldEqualIfNaNsEqual, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "metric",
				StartTime: retentionFrom,
				StopTime:  retentionUntil,
				StepTime:  retention, Values: []float64{0, 1, 2, 3},
			},
			},
			Metrics:  []string{metric},
			Patterns: []string{pattern},
		})
	})
}

func TestLocalSourceFetchWithMultiplePatterns(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	localSource := Create(database)
	defer mockCtrl.Finish()

	metricsTTL := int64(3600)

	retention1 := int64(10)
	pattern1 := pattern1
	metric1 := metric1
	dataList1 := map[string][]*moira.MetricValue{
		metric1: {
			{RetentionTimestamp: 20, Timestamp: 23, Value: 0.1},
			{RetentionTimestamp: 30, Timestamp: 33, Value: 1.1},
			{RetentionTimestamp: 40, Timestamp: 43, Value: 2.1},
			{RetentionTimestamp: 50, Timestamp: 53, Value: 3.1},
			{RetentionTimestamp: 60, Timestamp: 63, Value: 4.1},
			{RetentionTimestamp: 70, Timestamp: 73, Value: 5.1},
		},
	}

	retention2 := int64(20)
	pattern2 := pattern2
	metric2 := metric2
	dataList2 := map[string][]*moira.MetricValue{
		metric2: {
			{RetentionTimestamp: 0, Timestamp: 3, Value: 0.5},
			{RetentionTimestamp: 20, Timestamp: 23, Value: 1.5},
			{RetentionTimestamp: 40, Timestamp: 43, Value: 2.5},
			{RetentionTimestamp: 60, Timestamp: 63, Value: 3.5},
			{RetentionTimestamp: 80, Timestamp: 83, Value: 4.5},
		},
	}

	Convey("Test success evaluate", t, func() {
		database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL).AnyTimes()

		database.EXPECT().GetPatternMetrics(pattern1).Return([]string{metric1}, nil).AnyTimes()
		database.EXPECT().GetMetricRetention(metric1).Return(retention1, nil).AnyTimes()
		database.EXPECT().GetMetricsValues([]string{metric1}, gomock.Any(), gomock.Any()).Return(dataList1, nil).AnyTimes()

		database.EXPECT().GetPatternMetrics(pattern2).Return([]string{metric2}, nil).AnyTimes()
		database.EXPECT().GetMetricRetention(metric2).Return(retention2, nil).AnyTimes()
		database.EXPECT().GetMetricsValues([]string{metric2}, gomock.Any(), gomock.Any()).Return(dataList2, nil).AnyTimes().AnyTimes()

		result, err := localSource.Fetch("alias(sum(super.puper.pattern, super.duper.pattern), 'metric')", 17, 77, true)

		So(err, ShouldBeNil)
		So(result, shouldEqualIfNaNsEqual, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "metric",
				StartTime: 20,
				StopTime:  80,
				StepTime:  20,
				Values:    []float64{2.1, 5.1, 8.1},
			},
			},
			Metrics:  []string{metric1, metric2},
			Patterns: []string{pattern1, pattern2},
		})
	})
}

func TestLocalMetricsTTL(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	localSource := Create(dataBase)
	ttl := int64(42)

	Convey("Returns exact value from the database", t, func() {
		dataBase.EXPECT().GetMetricsTTLSeconds().Return(ttl)
		actual := localSource.GetMetricsTTLSeconds()
		So(actual, ShouldEqual, ttl)
	})
}

func TestLocal_IsConfigured(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	localSource := Create(dataBase)

	Convey("Always true", t, func() {
		actual, err := localSource.IsConfigured()
		So(err, ShouldBeNil)
		So(actual, ShouldBeTrue)
	})
}

func TestLocal_evalExpr(t *testing.T) {
	Convey("When everything is correct, we don't return any error", t, func() {
		ctx := evalCtx{from: time.Now().Add(-1 * time.Hour).Unix(), until: time.Now().Unix()}
		target := `seriesByTag('name=k8s.dev-cl1.kube_pod_status_ready', 'condition!=true', 'namespace=default', 'pod=~*')`

		expression, err := ctx.parse(target)
		So(err, ShouldBeNil)
		res, err := ctx.eval("target", expression, &fetchedMetrics{metricsMap: nil})
		So(err, ShouldBeNil)
		So(res, ShouldBeNil)
	})

	Convey("When get panic, it should return error", t, func() {
		ctx := evalCtx{from: 0, until: 0}

		expression, _ := ctx.parse(`;fg`)
		res, err := ctx.eval("target", expression, &fetchedMetrics{metricsMap: nil})
		So(err.Error(), ShouldContainSubstring, "panic while evaluate target target: message: 'runtime error: invalid memory address or nil pointer dereference")
		So(res, ShouldBeNil)
	})

	Convey("When no metrics, should not return error", t, func() {
		ctx := evalCtx{from: time.Now().Add(-1 * time.Hour).Unix(), until: time.Now().Unix()}
		target := `alias( divideSeries( alias( sumSeries( exclude( groupByNode( OFD.Production.{ofd-api,ofd-front}.*.fns-service-client.v120.*.GetCashboxRegistrationInformationAsync.ResponseCode.*.Meter.Rate-15-min-Requests-per-s, 9, "sum" ), "Ok" ) ), "bad" ), alias( sumSeries( OFD.Production.{ofd-api,ofd-front}.*.fns-service-client.v120.*.GetCashboxRegistrationInformationAsync.ResponseCode.*.Meter.Rate-15-min-Requests-per-s ), "total" ) ), "Result" )`

		expression, err := ctx.parse(target)
		So(err, ShouldBeNil)
		res, err := ctx.eval("target", expression, &fetchedMetrics{metricsMap: make(map[parser.MetricRequest][]*types.MetricData)})
		So(err, ShouldBeNil)
		So(res, ShouldBeEmpty)
	})

	Convey("When got unknown func, should return error", t, func() {
		ctx := evalCtx{from: time.Now().Add(-1 * time.Hour).Unix(), until: time.Now().Unix()}
		target := `vf('name=k8s.dev-cl1.kube_pod_status_ready', 'condition!=true', 'namespace=default', 'pod=~*')`

		expression, _ := ctx.parse(target)
		res, err := ctx.eval("target", expression, &fetchedMetrics{metricsMap: nil})
		So(err, ShouldBeError)
		So(err.Error(), ShouldResemble, `Unknown graphite function: "vf"`)
		So(res, ShouldBeNil)
	})
}

func shouldEqualIfNaNsEqual(actual interface{}, expected ...interface{}) string {
	allowUnexportedOption := cmp.AllowUnexported(types.MetricData{})

	floatOption := cmp.Comparer(func(a, b float64) bool {
		return math.IsNaN(a) && math.IsNaN(b) || a == b
	})
	metricDataOption := cmp.Comparer(func(a, b *types.MetricData) bool {
		return cmp.Equal(*a, *b, floatOption, allowUnexportedOption)
	})

	return cmp.Diff(
		actual,
		expected[0],
		floatOption,
		metricDataOption,
		allowUnexportedOption,
	)
}
