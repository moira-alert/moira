package target

import (
	"fmt"
	"github.com/go-graphite/carbonapi/expr"
	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func Test1(t *testing.T) {
	log, _ := logging.GetLogger("test")
	db := redis.NewDatabase(log, redis.Config{Host: "vm-moira-r1.dev.kontur.ru", Port: "6379"})

	now := time.Now().Unix()
	from := now - 600

	k, err := EvaluateTarget(db, "aliasByNode(reduceSeries(mapSeries(DevOps.system.vm-d*.memory.{MemAvailable,MemTotal}, 2), 'asPercent', 4, 'MemAvailable', 'MemTotal'), 2)", from, now, false)
	log.Info(k)
	log.Info(err)

	//k, _ := EvaluateTarget(db, "gofra.import.new", from, now, false)
	//log.Info(k)
}

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
			So(err, ShouldResemble, expr.ErrMissingExpr)
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
			So(err.Error(), ShouldResemble, "unknown function in evalExpr: \"aliasByNoe\"")
			So(result, ShouldBeNil)
		})
	})

	Convey("Test no metrics", t, func() {
		dataBase.EXPECT().GetPatternMetrics("super.puper.pattern").Return([]string{}, nil)
		result, err := EvaluateTarget(dataBase, "aliasByNode(super.puper.pattern, 2)", from, until, true)
		So(err, ShouldBeNil)
		So(result, ShouldResemble, &EvaluationResult{
			TimeSeries: make([]*TimeSeries, 0),
			Metrics:    make([]string, 0),
			Patterns:   []string{"super.puper.pattern"},
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
			TimeSeries: []*TimeSeries{&TimeSeries{FetchResponse: fetchResponse}},
			Metrics:    []string{metric},
			Patterns:   []string{"super.puper.pattern"},
		})
	})
}
