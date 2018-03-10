package target

import (
	"fmt"
	"testing"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/go-graphite/carbonapi/pkg/parser"
	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEvaluateTarget(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
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
	metricErr := fmt.Errorf("Ooops, metric error")

	Convey("Errors tests", t, func() {
		Convey("Error while ParseExpr", func() {
			result, err := EvaluateTarget(dataBase, "", from, until, true)
			So(err, ShouldResemble, ErrParseExpr{target: "", internalError: parser.ErrMissingExpr})
			So(err.Error(), ShouldResemble, "failed to parse target '': missing expression")
			So(result, ShouldBeNil)
		})

		Convey("Error in fetch data", func() {
			dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
			dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
			dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(nil, metricErr)
			result, err := EvaluateTarget(dataBase, "super.puper.pattern", from, until, true)
			So(err, ShouldResemble, metricErr)
			So(result, ShouldBeNil)
		})

		Convey("Error evaluate target", func() {
			dataBase.EXPECT().GetPatternMetrics("super.puper.pattern").Return([]string{metric}, nil)
			dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
			dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(dataList, nil)
			result, err := EvaluateTarget(dataBase, "aliasByNoe(super.puper.pattern, 2)", from, until, true)
			So(err.Error(), ShouldResemble, "Unknown graphite function: \"aliasByNoe\"")
			So(result, ShouldBeNil)
		})
	})

	Convey("Test no metrics", t, func() {
		dataBase.EXPECT().GetPatternMetrics("super.puper.pattern").Return([]string{}, nil)
		result, err := EvaluateTarget(dataBase, "aliasByNode(super.puper.pattern, 2)", from, until, true)
		So(err, ShouldBeNil)
		fetchResponse := pb.FetchResponse{
			Name:      "pattern",
			StartTime: int32(from),
			StopTime:  int32(until),
			StepTime:  60,
			Values:    []float64{},
			IsAbsent:  []bool{},
		}
		So(result, ShouldResemble, &EvaluationResult{
			TimeSeries: []*TimeSeries{{
				MetricData: types.MetricData{FetchResponse: fetchResponse},
				Wildcard:   true,
			}},
			Metrics:  make([]string, 0),
			Patterns: []string{"super.puper.pattern"},
		})
	})

	Convey("Test success evaluate", t, func() {
		dataBase.EXPECT().GetPatternMetrics("super.puper.pattern").Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(dataList, nil)
		result, err := EvaluateTarget(dataBase, "aliasByNode(super.puper.pattern, 2)", from, until, true)
		fetchResponse := pb.FetchResponse{
			Name:      "metric",
			StartTime: int32(from),
			StopTime:  int32(until),
			StepTime:  int32(retention),
			Values:    []float64{0, 1, 2, 3, 4},
			IsAbsent:  make([]bool, 5),
		}
		So(err, ShouldBeNil)
		So(result, ShouldResemble, &EvaluationResult{
			TimeSeries: []*TimeSeries{{
				MetricData: types.MetricData{FetchResponse: fetchResponse},
			}},
			Metrics:  []string{metric},
			Patterns: []string{"super.puper.pattern"},
		})
	})
}
