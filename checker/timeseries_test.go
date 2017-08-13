package checker

import (
	"fmt"
	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"math"
	"testing"
)

func TestGetTimeSeries(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	pattern := "super.puper.pattern"
	addPattern := "additional.pattern"
	metric := "super.puper.metric"
	addMetric := "additional.metric"
	addMetric2 := "additional.metric2"
	metricValues := []*moira.MetricValue{
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
	}
	dataList := map[string][]*moira.MetricValue{
		metric: metricValues,
	}

	var from int64 = 17
	var until int64 = 67
	var retention int64 = 10
	metricErr := fmt.Errorf("Ooops, metric error")

	triggerChecker := &TriggerChecker{
		Database: dataBase,
		isSimple: true,
		trigger: &moira.Trigger{
			Targets: []string{pattern},
		},
	}

	Convey("Error test", t, func() {
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(nil, metricErr)
		actual, metrics, err := triggerChecker.getTimeSeries(from, until)
		So(actual, ShouldBeNil)
		So(metrics, ShouldBeNil)
		So(err, ShouldBeError)
		So(err, ShouldResemble, metricErr)
	})

	Convey("Test no metrics", t, func() {
		Convey("in main target", func() {
			dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{}, nil)
			actual, metrics, err := triggerChecker.getTimeSeries(from, until)
			expected := &triggerTimeSeries{
				Main:       make([]*TimeSeries, 0),
				Additional: make([]*TimeSeries, 0),
			}
			So(actual, ShouldResemble, expected)
			So(metrics, ShouldBeEmpty)
			So(err, ShouldBeNil)
		})

		triggerChecker.trigger.Targets = append(triggerChecker.trigger.Targets, addPattern)

		Convey("in additional target", func() {
			dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{}, nil)
			dataBase.EXPECT().GetPatternMetrics(addPattern).Return([]string{}, nil)
			actual, metrics, err := triggerChecker.getTimeSeries(from, until)
			So(err, ShouldBeError)
			So(err, ShouldResemble, fmt.Errorf("Target #2 has no timeseries"))
			So(actual, ShouldBeNil)
			So(metrics, ShouldBeNil)
		})
	})

	triggerChecker.trigger.Targets = []string{pattern}

	Convey("Test has metrics", t, func() {
		Convey("Only one target", func() {
			dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
			dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
			dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(dataList, nil)
			actual, metrics, err := triggerChecker.getTimeSeries(from, until)
			fetchResponse := pb.FetchResponse{
				Name:      metric,
				StartTime: int32(from),
				StopTime:  int32(until),
				StepTime:  int32(retention),
				Values:    []float64{0, 1, 2, 3, 4},
				IsAbsent:  make([]bool, 5, 5),
			}
			expected := &triggerTimeSeries{
				Main:       []*TimeSeries{{FetchResponse: fetchResponse}},
				Additional: make([]*TimeSeries, 0),
			}
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
			So(metrics, ShouldResemble, []string{metric})
		})

		Convey("Two targets", func() {
			triggerChecker.trigger.Targets = []string{pattern, addPattern}
			dataList[addMetric] = metricValues

			dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
			dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
			dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(dataList, nil)

			dataBase.EXPECT().GetPatternMetrics(addPattern).Return([]string{addMetric}, nil)
			dataBase.EXPECT().GetMetricRetention(addMetric).Return(retention, nil)
			dataBase.EXPECT().GetMetricsValues([]string{addMetric}, from, until).Return(dataList, nil)

			actual, metrics, err := triggerChecker.getTimeSeries(from, until)
			fetchResponse := pb.FetchResponse{
				Name:      metric,
				StartTime: int32(from),
				StopTime:  int32(until),
				StepTime:  int32(retention),
				Values:    []float64{0, 1, 2, 3, 4},
				IsAbsent:  make([]bool, 5, 5),
			}
			addFetchResponse := fetchResponse
			addFetchResponse.Name = addMetric
			expected := &triggerTimeSeries{
				Main:       []*TimeSeries{{FetchResponse: fetchResponse}},
				Additional: []*TimeSeries{{FetchResponse: addFetchResponse}},
			}

			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
			So(metrics, ShouldResemble, []string{metric, addMetric})
		})

		Convey("Two targets with many metrics in additional target", func() {
			dataList[addMetric2] = metricValues

			dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
			dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
			dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(dataList, nil)

			dataBase.EXPECT().GetPatternMetrics(addPattern).Return([]string{addMetric, addMetric2}, nil)
			dataBase.EXPECT().GetMetricRetention(addMetric).Return(retention, nil)
			dataBase.EXPECT().GetMetricsValues([]string{addMetric, addMetric2}, from, until).Return(dataList, nil)

			actual, metrics, err := triggerChecker.getTimeSeries(from, until)
			So(err, ShouldBeError)
			So(err, ShouldResemble, fmt.Errorf("Target #2 has more than one timeseries"))
			So(actual, ShouldBeNil)
			So(metrics, ShouldBeNil)
		})
	})
}

func TestGetTargetName(t *testing.T) {
	tts := triggerTimeSeries{}

	Convey("GetMainTargetName", t, func() {
		So(tts.getMainTargetName(), ShouldResemble, "t1")
	})

	Convey("GetAdditionalTargetName", t, func() {
		for i := 0; i < 5; i++ {
			So(tts.getAdditionalTargetName(i), ShouldResemble, fmt.Sprintf("t%v", i+2))
		}
	})
}

func TestGetTimestampValue(t *testing.T) {
	Convey("IsAbsent only false", t, func() {
		fetchResponse := pb.FetchResponse{
			Name:      "m",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{0, 1, 2, 3, 4},
			IsAbsent:  []bool{false, false, false, false, false},
		}
		timeSeries := TimeSeries{FetchResponse: fetchResponse}
		Convey("Has value", func() {
			actual := timeSeries.getTimestampValue(18)
			So(actual, ShouldEqual, 0)
			actual = timeSeries.getTimestampValue(17)
			So(actual, ShouldEqual, 0)
			actual = timeSeries.getTimestampValue(24)
			So(actual, ShouldEqual, 0)
			actual = timeSeries.getTimestampValue(36)
			So(actual, ShouldEqual, 1)
			actual = timeSeries.getTimestampValue(37)
			So(actual, ShouldEqual, 2)
			actual = timeSeries.getTimestampValue(66)
			So(actual, ShouldEqual, 4)
		})

		Convey("No value", func() {
			actual := timeSeries.getTimestampValue(16)
			So(math.IsNaN(actual), ShouldBeTrue)
			actual = timeSeries.getTimestampValue(67)
			So(math.IsNaN(actual), ShouldBeTrue)
		})
	})

	Convey("IsAbsent has true", t, func() {
		fetchResponse := pb.FetchResponse{
			Name:      "m",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{0, 1, 2, 3, 4},
			IsAbsent:  []bool{false, true, true, false, true},
		}
		timeSeries := TimeSeries{FetchResponse: fetchResponse}

		actual := timeSeries.getTimestampValue(18)
		So(actual, ShouldEqual, 0)
		actual = timeSeries.getTimestampValue(27)
		So(math.IsNaN(actual), ShouldBeTrue)
		actual = timeSeries.getTimestampValue(30)
		So(math.IsNaN(actual), ShouldBeTrue)
		actual = timeSeries.getTimestampValue(39)
		So(math.IsNaN(actual), ShouldBeTrue)
		actual = timeSeries.getTimestampValue(49)
		So(actual, ShouldEqual, 3)
		actual = timeSeries.getTimestampValue(57)
		So(math.IsNaN(actual), ShouldBeTrue)
		actual = timeSeries.getTimestampValue(66)
		So(math.IsNaN(actual), ShouldBeTrue)
	})
}

func TestGetExpressionValues(t *testing.T) {
	Convey("Has only main timeSeries", t, func() {
		fetchResponse := pb.FetchResponse{
			Name:      "m",
			StartTime: int32(17),
			StopTime:  int32(67),
			StepTime:  int32(10),
			Values:    []float64{0, 1, 2, 3, 4},
			IsAbsent:  []bool{false, true, true, false, true},
		}
		timeSeries := TimeSeries{FetchResponse: fetchResponse}
		tts := &triggerTimeSeries{
			Main: []*TimeSeries{&timeSeries},
		}

		values, noEmptyValues := tts.getExpressionValues(&timeSeries, 17)
		So(noEmptyValues, ShouldBeTrue)
		So(values, ShouldResemble, ExpressionValues(map[string]float64{"t1": 0}))

		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 67)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, ExpressionValues(make(map[string]float64)))

		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 11)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, ExpressionValues(make(map[string]float64)))

		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 44)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, ExpressionValues(make(map[string]float64)))

		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 53)
		So(noEmptyValues, ShouldBeTrue)
		So(values, ShouldResemble, ExpressionValues(map[string]float64{"t1": 3}))
	})

	Convey("Has additional series", t, func() {
		fetchResponse := pb.FetchResponse{
			Name:      "main",
			StartTime: int32(17),
			StopTime:  int32(67),
			StepTime:  int32(10),
			Values:    []float64{0, 1, 2, 3, 4},
			IsAbsent:  []bool{false, true, true, false, true},
		}
		timeSeries := TimeSeries{FetchResponse: fetchResponse}
		fetchResponseAdd := pb.FetchResponse{
			Name:      "main",
			StartTime: int32(17),
			StopTime:  int32(67),
			StepTime:  int32(10),
			Values:    []float64{4, 3, 2, 1, 0},
			IsAbsent:  []bool{false, false, true, true, false},
		}
		timeSeriesAdd := TimeSeries{FetchResponse: fetchResponseAdd}
		tts := &triggerTimeSeries{
			Main:       []*TimeSeries{&timeSeries},
			Additional: []*TimeSeries{&timeSeriesAdd},
		}

		values, noEmptyValues := tts.getExpressionValues(&timeSeries, 17)
		So(noEmptyValues, ShouldBeTrue)
		So(values, ShouldResemble, ExpressionValues(map[string]float64{"t1": 0, "t2": 4}))

		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 29)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, ExpressionValues(make(map[string]float64)))

		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 42)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, ExpressionValues(make(map[string]float64)))

		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 50)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, ExpressionValues(map[string]float64{"t1": 3}))

		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 65)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, ExpressionValues(make(map[string]float64)))
	})
}
