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
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	functions.New(make(map[string]string))
}

func TestEvaluateTarget(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	localSource := Create(dataBase)
	defer mockCtrl.Finish()

	pattern := "super.puper.pattern"
	metric := "super.puper.metric"
	dataList := map[string][]*moira.MetricValue{
		metric: {
			{
				RetentionTimestamp: 20,
				Timestamp:          23,
				Value:              0,
			},
			{
				RetentionTimestamp: 30,
				Timestamp:          33,
				Value:              1,
			},
			{
				RetentionTimestamp: 40,
				Timestamp:          43,
				Value:              2,
			},
			{
				RetentionTimestamp: 50,
				Timestamp:          53,
				Value:              3,
			},
			{
				RetentionTimestamp: 60,
				Timestamp:          63,
				Value:              4,
			},
		},
	}

	var from int64 = 17
	var until int64 = 67
	var retention int64 = 10
	var metricsTTL int64 = 3600
	metricErr := fmt.Errorf("Ooops, metric error")

	Convey("Errors tests", t, func() {
		Convey("Error while ParseExpr", func() {
			dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)
			result, err := localSource.Fetch("", from, until, true)
			So(err, ShouldResemble, ErrParseExpr{target: "", internalError: parser.ErrMissingExpr})
			So(err.Error(), ShouldResemble, "failed to parse target '': missing expression")
			So(result, ShouldBeNil)
		})

		Convey("Error in fetch data", func() {
			dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
			dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
			dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(nil, metricErr)
			dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)
			result, err := localSource.Fetch("super.puper.pattern", from, until, true)
			So(err, ShouldResemble, metricErr)
			So(result, ShouldBeNil)
		})

		Convey("Error evaluate target", func() {
			dataBase.EXPECT().GetPatternMetrics("super.puper.pattern").Return([]string{metric}, nil)
			dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
			dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(dataList, nil)
			dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)
			result, err := localSource.Fetch("aliasByNoe(super.puper.pattern, 2)", from, until, true)
			So(err.Error(), ShouldResemble, "failed to evaluate target 'aliasByNoe(super.puper.pattern, 2)': unknown function in evalExpr: \"aliasByNoe\"")
			So(result, ShouldBeNil)
		})

		Convey("Panic while evaluate target", func() {
			dataBase.EXPECT().GetPatternMetrics("super.puper.pattern").Return([]string{metric}, nil)
			dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
			dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(dataList, nil)
			dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)
			result, err := localSource.Fetch("movingAverage(super.puper.pattern, -1)", from, until, true)
			expectedErrSubstring := strings.Split(ErrEvaluateTargetFailedWithPanic{target: "movingAverage(super.puper.pattern, -1)"}.Error(), ":")[0]
			So(err.Error(), ShouldStartWith, expectedErrSubstring)
			So(result, ShouldBeNil)
		})
	})

	Convey("Test no metrics", t, func() {
		dataBase.EXPECT().GetPatternMetrics("super.puper.pattern").Return([]string{}, nil)
		dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)
		result, err := localSource.Fetch("aliasByNode(super.puper.pattern, 2)", from, until, true)
		So(err, ShouldBeNil)
		So(result, ShouldResemble, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "pattern",
				StartTime: from,
				StopTime:  until,
				StepTime:  60,
				Values:    []float64{},
				Wildcard:  true,
			}},
			Metrics:  make([]string, 0),
			Patterns: []string{"super.puper.pattern"},
		})
	})

	Convey("Test success evaluate", t, func() {
		dataBase.EXPECT().GetPatternMetrics("super.puper.pattern").Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(dataList, nil)
		dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)
		result, err := localSource.Fetch("aliasByNode(super.puper.pattern, 2)", from, until, true)
		So(err, ShouldBeNil)
		So(result, ShouldResemble, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "metric",
				StartTime: from,
				StopTime:  until,
				StepTime:  retention,
				Values:    []float64{0, 1, 2, 3, 4},
			},
			},
			Metrics:  []string{metric},
			Patterns: []string{"super.puper.pattern"},
		})
	})

	Convey("Test success evaluate multiple metrics with pow function", t, func() {
		metrics := []string{
			"apps.server1.process.cpu.usage",
			"apps.server2.process.cpu.usage",
			"apps.server3.process.cpu.usage",
		}

		multipleDataList := make(map[string][]*moira.MetricValue)
		multipleDataList["apps.server1.process.cpu.usage"] = []*moira.MetricValue{
			{
				RetentionTimestamp: 20,
				Timestamp:          23,
				Value:              0.5,
			},
			{
				RetentionTimestamp: 30,
				Timestamp:          33,
				Value:              0.4,
			},
			{
				RetentionTimestamp: 40,
				Timestamp:          43,
				Value:              0.5,
			},
			{
				RetentionTimestamp: 50,
				Timestamp:          53,
				Value:              0.5,
			},
			{
				RetentionTimestamp: 60,
				Timestamp:          63,
				Value:              0.5,
			},
		}
		multipleDataList["apps.server2.process.cpu.usage"] = []*moira.MetricValue{
			{
				RetentionTimestamp: 20,
				Timestamp:          23,
				Value:              math.NaN(),
			},
			{
				RetentionTimestamp: 30,
				Timestamp:          33,
				Value:              math.NaN(),
			},
			{
				RetentionTimestamp: 40,
				Timestamp:          43,
				Value:              math.NaN(),
			},
			{
				RetentionTimestamp: 50,
				Timestamp:          53,
				Value:              math.NaN(),
			},
			{
				RetentionTimestamp: 60,
				Timestamp:          63,
				Value:              math.NaN(),
			},
		}
		multipleDataList["apps.server3.process.cpu.usage"] = []*moira.MetricValue{
			{
				RetentionTimestamp: 20,
				Timestamp:          23,
				Value:              0.5,
			},
			{
				RetentionTimestamp: 30,
				Timestamp:          33,
				Value:              0.5,
			},
			{
				RetentionTimestamp: 40,
				Timestamp:          43,
				Value:              0.5,
			},
			{
				RetentionTimestamp: 50,
				Timestamp:          53,
				Value:              0.4,
			},
			{
				RetentionTimestamp: 60,
				Timestamp:          63,
				Value:              0.5,
			},
		}

		dataBase.EXPECT().GetPatternMetrics("apps.*.process.cpu.usage").Return(metrics, nil)
		dataBase.EXPECT().GetMetricRetention(metrics[0]).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues(metrics, gomock.Any(), until).Return(multipleDataList, nil)
		dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)

		result, err := localSource.Fetch("alias(sumSeries(pow(apps.*.process.cpu.usage, 0)), 'alive replicas')", from, until, true)
		So(err, ShouldBeNil)
		So(result, ShouldResemble, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "alive replicas",
				StartTime: from,
				StopTime:  until,
				StepTime:  retention,
				Values:    []float64{2, 2, 2, 2, 2},
			},
			},
			Metrics:  metrics,
			Patterns: []string{"apps.*.process.cpu.usage"},
		})
	})

	Convey("Test enormous fetch interval", t, func() {
		var fromLongAgo int64 = 0
		var untilDistantFuture int64 = 1e15
		var ttl = retention - 1
		var distantFutureDataList = map[string][]*moira.MetricValue{
			metric: {
				{
					RetentionTimestamp: untilDistantFuture,
					Timestamp:          untilDistantFuture,
					Value:              0,
				},
			},
		}

		dataBase.EXPECT().GetPatternMetrics("super.puper.pattern").Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, untilDistantFuture-ttl, untilDistantFuture).Return(distantFutureDataList, nil)
		dataBase.EXPECT().GetMetricsTTLSeconds().Return(ttl)
		result, err := localSource.Fetch("aliasByNode(super.puper.pattern, 2)", fromLongAgo, untilDistantFuture, true)
		So(err, ShouldBeNil)
		So(result, ShouldResemble, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "metric",
				StartTime: untilDistantFuture - ttl,
				StopTime:  untilDistantFuture,
				StepTime:  retention,
				Values:    []float64{0},
			},
			},
			Metrics:  []string{metric},
			Patterns: []string{"super.puper.pattern"},
		})
	})

	Convey("Test success evaluate pipe target", t, func() {
		dataBase.EXPECT().GetPatternMetrics("super.puper.pattern").Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(dataList, nil)
		dataBase.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL)
		result, err := localSource.Fetch("super.puper.pattern | scale(100) | aliasByNode(2)", from, until, true)
		So(err, ShouldBeNil)
		So(result, ShouldResemble, &FetchResult{
			MetricsData: []metricSource.MetricData{{
				Name:      "metric",
				StartTime: from,
				StopTime:  until,
				StepTime:  retention,
				Values:    []float64{0, 100, 200, 300, 400},
			}},
			Metrics:  []string{metric},
			Patterns: []string{"super.puper.pattern"},
		})
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
		expression, _, err := parser.ParseExpr(`seriesByTag('name=k8s.dev-cl1.kube_pod_status_ready', 'condition!=true', 'namespace=default', 'pod=~*')`)
		So(err, ShouldBeNil)
		res, err := evalExpr("target", expression, time.Now().Add(-1*time.Hour).Unix(), time.Now().Unix(), nil)
		So(err, ShouldBeNil)
		So(res, ShouldBeNil)
	})

	Convey("When get panic, it should return error", t, func() {
		expression, _, _ := parser.ParseExpr(`;fg`)
		res, err := evalExpr("target", expression, 0, 0, nil)
		So(err.Error(), ShouldContainSubstring, "panic while evaluate target target: message: 'runtime error: invalid memory address or nil pointer dereference")
		So(res, ShouldBeNil)
	})

	Convey("When no metrics, should not return error", t, func() {
		expression, _, err := parser.ParseExpr(`alias( divideSeries( alias( sumSeries( exclude( groupByNode( OFD.Production.{ofd-api,ofd-front}.*.fns-service-client.v120.*.GetCashboxRegistrationInformationAsync.ResponseCode.*.Meter.Rate-15-min-Requests-per-s, 9, "sum" ), "Ok" ) ), "bad" ), alias( sumSeries( OFD.Production.{ofd-api,ofd-front}.*.fns-service-client.v120.*.GetCashboxRegistrationInformationAsync.ResponseCode.*.Meter.Rate-15-min-Requests-per-s ), "total" ) ), "Result" )`)
		So(err, ShouldBeNil)
		res, err := evalExpr("target", expression, time.Now().Add(-1*time.Hour).Unix(), time.Now().Unix(), make(map[parser.MetricRequest][]*types.MetricData))
		So(err, ShouldBeNil)
		So(res, ShouldBeEmpty)
	})

	Convey("When got unknown func, should return error", t, func() {
		expression, _, _ := parser.ParseExpr(`vf('name=k8s.dev-cl1.kube_pod_status_ready', 'condition!=true', 'namespace=default', 'pod=~*')`)
		res, err := evalExpr("target", expression, time.Now().Add(-1*time.Hour).Unix(), time.Now().Unix(), nil)
		So(err, ShouldBeError)
		So(err.Error(), ShouldResemble, `failed to evaluate target 'target': unknown function in evalExpr: "vf"`)
		So(res, ShouldBeNil)
	})
}
