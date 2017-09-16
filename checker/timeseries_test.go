package checker

import (
	"fmt"
	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/expression"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/moira-alert/moira/target"
	. "github.com/smartystreets/goconvey/convey"
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
		trigger: &moira.Trigger{
			Targets:  []string{pattern},
			Patterns: []string{pattern},
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
				Main:       make([]*target.TimeSeries, 0),
				Additional: make([]*target.TimeSeries, 0),
			}
			So(actual, ShouldResemble, expected)
			So(metrics, ShouldBeEmpty)
			So(err, ShouldBeNil)
		})
	})

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
				IsAbsent:  make([]bool, 5),
			}
			expected := &triggerTimeSeries{
				Main:       []*target.TimeSeries{{FetchResponse: fetchResponse}},
				Additional: make([]*target.TimeSeries, 0),
			}
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
			So(metrics, ShouldResemble, []string{metric})
		})

		Convey("Two targets", func() {
			triggerChecker.trigger.Targets = []string{pattern, addPattern}
			triggerChecker.trigger.Patterns = []string{pattern, addPattern}
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
				Values:    []float64{0, 1, 2, 3},
				IsAbsent:  make([]bool, 4),
			}
			addFetchResponse := fetchResponse
			addFetchResponse.Name = addMetric
			expected := &triggerTimeSeries{
				Main:       []*target.TimeSeries{{FetchResponse: fetchResponse}},
				Additional: []*target.TimeSeries{{FetchResponse: addFetchResponse}},
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

func TestGetExpressionValues(t *testing.T) {
	Convey("Has only main timeSeries", t, func() {
		fetchResponse := pb.FetchResponse{
			Name:      "m",
			StartTime: int32(17),
			StopTime:  int32(67),
			StepTime:  int32(10),
			Values:    []float64{0.0, 1.0, 2.0, 3.0, 4.0},
			IsAbsent:  []bool{false, true, true, false, true},
		}
		timeSeries := target.TimeSeries{FetchResponse: fetchResponse}
		tts := &triggerTimeSeries{
			Main: []*target.TimeSeries{&timeSeries},
		}
		expectedExpressionValues := expression.TriggerExpression{
			AdditionalTargetsValues: make(map[string]float64),
		}

		values, noEmptyValues := tts.getExpressionValues(&timeSeries, 17)
		So(noEmptyValues, ShouldBeTrue)
		So(values, ShouldResemble, expectedExpressionValues)

		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 67)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, expectedExpressionValues)

		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 11)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, expectedExpressionValues)

		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 44)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, expectedExpressionValues)

		expectedExpressionValues.MainTargetValue = 3
		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 53)
		So(noEmptyValues, ShouldBeTrue)
		So(values, ShouldResemble, expectedExpressionValues)
	})

	Convey("Has additional series", t, func() {
		fetchResponse := pb.FetchResponse{
			Name:      "main",
			StartTime: int32(17),
			StopTime:  int32(67),
			StepTime:  int32(10),
			Values:    []float64{0.0, 1.0, 2.0, 3.0, 4.0},
			IsAbsent:  []bool{false, true, true, false, true},
		}
		timeSeries := target.TimeSeries{FetchResponse: fetchResponse}
		fetchResponseAdd := pb.FetchResponse{
			Name:      "main",
			StartTime: int32(17),
			StopTime:  int32(67),
			StepTime:  int32(10),
			Values:    []float64{4.0, 3.0, 2.0, 1.0, 0.0},
			IsAbsent:  []bool{false, false, true, true, false},
		}
		timeSeriesAdd := target.TimeSeries{FetchResponse: fetchResponseAdd}
		tts := &triggerTimeSeries{
			Main:       []*target.TimeSeries{&timeSeries},
			Additional: []*target.TimeSeries{&timeSeriesAdd},
		}

		expectedExpressionValues := expression.TriggerExpression{
			AdditionalTargetsValues: make(map[string]float64),
		}

		values, noEmptyValues := tts.getExpressionValues(&timeSeries, 29)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, expectedExpressionValues)

		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 42)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, expectedExpressionValues)

		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 65)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, expectedExpressionValues)

		expectedExpressionValues.MainTargetValue = 3
		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 50)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, expectedExpressionValues)

		expectedExpressionValues.MainTargetValue = 0
		expectedExpressionValues.AdditionalTargetsValues["t2"] = 4
		values, noEmptyValues = tts.getExpressionValues(&timeSeries, 17)
		So(noEmptyValues, ShouldBeTrue)
		So(values, ShouldResemble, expectedExpressionValues)
	})
}
