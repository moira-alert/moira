package local

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-graphite/carbonapi/expr/functions"
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
			So(err.Error(), ShouldResemble, "Unknown graphite function: \"aliasByNoe\"")
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
