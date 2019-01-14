package checker

import (
	"fmt"
	"math"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/expression"
	"github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetTimeSeries(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	pattern := "super.puper.pattern"
	metric := "super.puper.metric"

	pattern2 := "super.duper.pattern"
	metric2 := "super.duper.metric"

	addPattern := "additional.pattern"
	addMetric := "additional.metric"
	addMetric2 := "additional.metric2"

	oneMorePattern := "one.more.pattern"
	oneMoreMetric1 := "one.more.metric.one"
	oneMoreMetric2 := "one.more.metric.two"

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
	metricErr := fmt.Errorf("ooops, metric error")

	triggerChecker := &TriggerChecker{
		Database: dataBase,
		Source:   local.CreateLocalSource(dataBase),
		trigger: &moira.Trigger{
			Targets:  []string{pattern},
			Patterns: []string{pattern},
		},
	}

	Convey("Error test", t, func() {
		dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
		dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
		dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(nil, metricErr)
		actual, metrics, err := triggerChecker.getFetchResult(from, until)
		So(actual, ShouldBeNil)
		So(metrics, ShouldBeNil)
		So(err, ShouldBeError)
		So(err, ShouldResemble, metricErr)
	})

	Convey("Test no metrics", t, func() {
		Convey("in main target", func() {
			dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{}, nil)
			actual, metrics, err := triggerChecker.getFetchResult(from, until)
			metricData := &metricSource.MetricData{
				Name:      pattern,
				StartTime: from,
				StopTime:  until,
				StepTime:  60,
				Values:    []float64{},
				Wildcard:  true,
			}
			expected := metricSource.MakeTriggerMetricsData([]*metricSource.MetricData{metricData}, make([]*metricSource.MetricData, 0))
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
			actual, metrics, err := triggerChecker.getFetchResult(from, until)
			metricData := &metricSource.MetricData{
				Name:      metric,
				StartTime: from,
				StopTime:  until,
				StepTime:  retention,
				Values:    []float64{0, 1, 2, 3, 4},
			}
			expected := metricSource.MakeTriggerMetricsData([]*metricSource.MetricData{metricData}, make([]*metricSource.MetricData, 0))
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

			actual, metrics, err := triggerChecker.getFetchResult(from, until)
			metricData := metricSource.MetricData{
				Name:      metric,
				StartTime: from,
				StopTime:  until,
				StepTime:  retention,
				Values:    []float64{0, 1, 2, 3},
			}
			addMetricData := metricData
			addMetricData.Name = addMetric
			expected := metricSource.MakeTriggerMetricsData([]*metricSource.MetricData{&metricData}, []*metricSource.MetricData{&addMetricData})

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

			actual, metrics, err := triggerChecker.getFetchResult(from, until)
			So(err, ShouldBeError)
			So(err, ShouldResemble, ErrWrongTriggerTargets([]int{2}))
			So(err.Error(), ShouldResemble, "Target t2 has more than one timeseries")
			So(actual, ShouldBeNil)
			So(metrics, ShouldBeNil)
		})

		Convey("Four targets with many metrics in additional targets", func() {
			triggerChecker.trigger.Targets = []string{pattern, addPattern, pattern2, oneMorePattern}
			triggerChecker.trigger.Patterns = []string{pattern, addPattern, pattern2, oneMorePattern}

			dataList[addMetric2] = metricValues
			dataList[metric2] = metricValues
			dataList[oneMoreMetric1] = metricValues
			dataList[oneMoreMetric2] = metricValues

			dataBase.EXPECT().GetPatternMetrics(pattern).Return([]string{metric}, nil)
			dataBase.EXPECT().GetMetricRetention(metric).Return(retention, nil)
			dataBase.EXPECT().GetMetricsValues([]string{metric}, from, until).Return(dataList, nil)

			dataBase.EXPECT().GetPatternMetrics(addPattern).Return([]string{addMetric, addMetric2}, nil)
			dataBase.EXPECT().GetMetricRetention(addMetric).Return(retention, nil)
			dataBase.EXPECT().GetMetricsValues([]string{addMetric, addMetric2}, from, until).Return(dataList, nil)

			dataBase.EXPECT().GetPatternMetrics(pattern2).Return([]string{metric2}, nil)
			dataBase.EXPECT().GetMetricRetention(metric2).Return(retention, nil)
			dataBase.EXPECT().GetMetricsValues([]string{metric2}, from, until).Return(dataList, nil)

			dataBase.EXPECT().GetPatternMetrics(oneMorePattern).Return([]string{oneMoreMetric1, oneMoreMetric2}, nil)
			dataBase.EXPECT().GetMetricRetention(oneMoreMetric1).Return(retention, nil)
			dataBase.EXPECT().GetMetricsValues([]string{oneMoreMetric1, oneMoreMetric2}, from, until).Return(dataList, nil)

			actual, metrics, err := triggerChecker.getFetchResult(from, until)
			So(err, ShouldBeError)
			So(err, ShouldResemble, ErrWrongTriggerTargets([]int{2, 4}))
			So(err.Error(), ShouldResemble, "Targets t2, t4 has more than one timeseries")
			So(actual, ShouldBeNil)
			So(metrics, ShouldBeNil)
		})
	})
}

func TestGetExpressionValues(t *testing.T) {
	Convey("Has only main timeSeries", t, func() {
		metricData := &metricSource.MetricData{
			Name:      "m",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{0.0, math.NaN(), math.NaN(), 3.0, math.NaN()},
		}
		tts := &metricSource.TriggerMetricsData{
			Main: []*metricSource.MetricData{metricData},
		}
		expectedExpressionValues := &expression.TriggerExpression{
			AdditionalTargetsValues: make(map[string]float64),
		}

		values, noEmptyValues := getExpressionValues(tts, metricData, 17)
		So(noEmptyValues, ShouldBeTrue)
		So(values, ShouldResemble, expectedExpressionValues)

		values, noEmptyValues = getExpressionValues(tts, metricData, 67)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, expectedExpressionValues)

		values, noEmptyValues = getExpressionValues(tts, metricData, 11)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, expectedExpressionValues)

		values, noEmptyValues = getExpressionValues(tts, metricData, 44)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, expectedExpressionValues)

		expectedExpressionValues.MainTargetValue = 3
		values, noEmptyValues = getExpressionValues(tts, metricData, 53)
		So(noEmptyValues, ShouldBeTrue)
		So(values, ShouldResemble, expectedExpressionValues)
	})

	Convey("Has additional series", t, func() {
		metricData := &metricSource.MetricData{
			Name:      "main",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{0.0, math.NaN(), math.NaN(), 3.0, math.NaN()},
		}
		metricDataAdd := &metricSource.MetricData{
			Name:      "main",
			StartTime: 17,
			StopTime:  67,
			StepTime:  10,
			Values:    []float64{4.0, 3.0, math.NaN(), math.NaN(), 0.0},
		}
		tts := &metricSource.TriggerMetricsData{
			Main:       []*metricSource.MetricData{metricData},
			Additional: []*metricSource.MetricData{metricDataAdd},
		}

		expectedExpressionValues := &expression.TriggerExpression{
			AdditionalTargetsValues: make(map[string]float64),
		}

		values, noEmptyValues := getExpressionValues(tts, metricData, 29)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, expectedExpressionValues)

		values, noEmptyValues = getExpressionValues(tts, metricData, 42)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, expectedExpressionValues)

		values, noEmptyValues = getExpressionValues(tts, metricData, 65)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, expectedExpressionValues)

		expectedExpressionValues.MainTargetValue = 3
		values, noEmptyValues = getExpressionValues(tts, metricData, 50)
		So(noEmptyValues, ShouldBeFalse)
		So(values, ShouldResemble, expectedExpressionValues)

		expectedExpressionValues.MainTargetValue = 0
		expectedExpressionValues.AdditionalTargetsValues["t2"] = 4
		values, noEmptyValues = getExpressionValues(tts, metricData, 17)
		So(noEmptyValues, ShouldBeTrue)
		So(values, ShouldResemble, expectedExpressionValues)
	})
}
