package checker

import (
	"fmt"
	"math"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/expression"
	metricSource "github.com/moira-alert/moira/metric_source"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFetchTriggerMetrics(t *testing.T) {

	Convey("Test fetch trigger metrics", t, func(c C) {
		mockCtrl := gomock.NewController(t)
		source := mock_metric_source.NewMockMetricSource(mockCtrl)
		fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
		defer mockCtrl.Finish()

		var from int64 = 17
		var until int64 = 67
		pattern := "super.puper.pattern"

		triggerChecker := &TriggerChecker{
			source: source,
			from:   from,
			until:  until,
			trigger: &moira.Trigger{
				Targets:  []string{pattern},
				Patterns: []string{pattern},
			},
			lastCheck: &moira.CheckData{
				Metrics: map[string]moira.MetricState{},
			},
		}

		Convey("no metrics in last check", t, func(c C) {
			Convey("fetch returns wildcard", t, func(c C) {
				source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil)
				fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{{Name: pattern, Wildcard: true}})
				fetchResult.EXPECT().GetPatternMetrics().Return([]string{}, nil)

				actual, err := triggerChecker.fetchTriggerMetrics()
				c.So(err, ShouldResemble, ErrTriggerHasOnlyWildcards{})
				c.So(actual, ShouldResemble, metricSource.MakeTriggerMetricsData([]*metricSource.MetricData{{Name: pattern, Wildcard: true}}, []*metricSource.MetricData{}))
			})

			Convey("fetch returns no metrics", t, func(c C) {
				source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil)
				fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{})
				fetchResult.EXPECT().GetPatternMetrics().Return([]string{}, nil)

				actual, err := triggerChecker.fetchTriggerMetrics()
				c.So(err, ShouldResemble, ErrTriggerHasNoMetrics{})
				c.So(actual, ShouldResemble, metricSource.MakeTriggerMetricsData([]*metricSource.MetricData{}, []*metricSource.MetricData{}))
			})
		})

		Convey("has metrics in last check", t, func(c C) {
			triggerChecker.lastCheck.Metrics["metric"] = moira.MetricState{}
			Convey("fetch returns wildcard", t, func(c C) {
				source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil)
				fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{{Name: pattern, Wildcard: true}})
				fetchResult.EXPECT().GetPatternMetrics().Return([]string{}, nil)

				actual, err := triggerChecker.fetchTriggerMetrics()
				c.So(err, ShouldBeEmpty)
				c.So(actual, ShouldResemble, metricSource.MakeTriggerMetricsData([]*metricSource.MetricData{{Name: pattern, Wildcard: true}}, []*metricSource.MetricData{}))
			})

			Convey("fetch returns no metrics", t, func(c C) {
				source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil)
				fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{})
				fetchResult.EXPECT().GetPatternMetrics().Return([]string{}, nil)

				actual, err := triggerChecker.fetchTriggerMetrics()
				c.So(err, ShouldBeEmpty)
				c.So(actual, ShouldResemble, metricSource.MakeTriggerMetricsData([]*metricSource.MetricData{}, []*metricSource.MetricData{}))
			})
		})
	})
}

func TestGetTimeSeries(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	source := mock_metric_source.NewMockMetricSource(mockCtrl)
	fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
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

	var from int64 = 17
	var until int64 = 67
	var retention int64 = 10

	triggerChecker := &TriggerChecker{
		database: dataBase,
		source:   source,
		from:     from,
		until:    until,
		trigger: &moira.Trigger{
			Targets:  []string{pattern},
			Patterns: []string{pattern},
		},
	}

	Convey("Error test", t, func(c C) {
		metricErr := fmt.Errorf("ooops, metric error")
		source.EXPECT().Fetch(pattern, from, until, true).Return(nil, metricErr)
		actual, metrics, err := triggerChecker.fetch()
		c.So(actual, ShouldBeNil)
		c.So(metrics, ShouldBeNil)
		c.So(err, ShouldBeError)
		c.So(err, ShouldResemble, metricErr)
	})

	Convey("Test no metrics", t, func(c C) {
		Convey("In main target", t, func(c C) {
			metricData := &metricSource.MetricData{
				Name:      pattern,
				StartTime: from,
				StopTime:  until,
				StepTime:  60,
				Values:    []float64{},
				Wildcard:  true,
			}

			source.EXPECT().Fetch(pattern, from, until, true).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{metricData})
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{}, nil)
			actual, metrics, err := triggerChecker.fetch()
			c.So(actual, ShouldResemble, metricSource.MakeTriggerMetricsData([]*metricSource.MetricData{metricData}, make([]*metricSource.MetricData, 0)))
			c.So(metrics, ShouldBeEmpty)
			c.So(err, ShouldBeNil)
		})

		Convey("In additional target", t, func(c C) {
			metricError := fmt.Errorf("metric error")
			triggerChecker1 := &TriggerChecker{
				database: dataBase,
				source:   source,
				from:     from,
				until:    until,
				trigger: &moira.Trigger{
					Targets:  []string{pattern, addPattern},
					Patterns: []string{pattern, addPattern},
				},
			}

			metricData := []*metricSource.MetricData{metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, from)}
			addMetricData := make([]*metricSource.MetricData, 0)

			source.EXPECT().Fetch(pattern, from, until, false).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return(metricData)
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)

			source.EXPECT().Fetch(addPattern, from, until, false).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return(addMetricData)

			Convey("get pattern metrics error", t, func(c C) {
				fetchResult.EXPECT().GetPatternMetrics().Return([]string{}, metricError)
				actual, metrics, err := triggerChecker1.fetch()
				c.So(actual, ShouldBeNil)
				c.So(metrics, ShouldBeNil)
				c.So(err, ShouldBeError)
				c.So(err, ShouldResemble, ErrTargetHasNoMetrics{targetIndex: 2})
			})

			Convey("get pattern metrics has metrics", t, func(c C) {
				fetchResult.EXPECT().GetPatternMetrics().Return([]string{addMetric}, nil)
				actual, metrics, err := triggerChecker1.fetch()
				c.So(actual, ShouldBeNil)
				c.So(metrics, ShouldBeNil)
				c.So(err, ShouldBeError)
				c.So(err, ShouldResemble, ErrTargetHasNoMetrics{targetIndex: 2})
				c.So(err.Error(), ShouldResemble, "target t3 has no timeseries")
			})

			Convey("get pattern metrics has no metrics", t, func(c C) {
				fetchResult.EXPECT().GetPatternMetrics().Return([]string{}, nil)
				actual, metrics, err := triggerChecker1.fetch()
				c.So(actual, ShouldResemble, metricSource.MakeTriggerMetricsData(metricData, []*metricSource.MetricData{nil}))
				c.So(metrics, ShouldResemble, []string{metric})
				c.So(err, ShouldBeNil)
			})
		})
	})

	Convey("Test has metrics", t, func(c C) {
		Convey("Only one target", t, func(c C) {
			source.EXPECT().Fetch(pattern, from, until, true).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return([]*metricSource.MetricData{metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, from)})
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)
			actual, metrics, err := triggerChecker.fetch()
			metricData := &metricSource.MetricData{
				Name:      metric,
				StartTime: from,
				StopTime:  until,
				StepTime:  retention,
				Values:    []float64{0, 1, 2, 3, 4},
			}
			expected := metricSource.MakeTriggerMetricsData([]*metricSource.MetricData{metricData}, make([]*metricSource.MetricData, 0))
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, expected)
			c.So(metrics, ShouldResemble, []string{metric})
		})

		Convey("Two targets", t, func(c C) {
			triggerChecker.trigger.Targets = []string{pattern, addPattern}
			triggerChecker.trigger.Patterns = []string{pattern, addPattern}

			metricData := []*metricSource.MetricData{metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, from)}
			addMetricData := []*metricSource.MetricData{metricSource.MakeMetricData(addMetric, []float64{0, 1, 2, 3, 4}, retention, from)}

			source.EXPECT().Fetch(pattern, from, until, false).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return(metricData)
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)

			source.EXPECT().Fetch(addPattern, from, until, false).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return(addMetricData)
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{addMetric}, nil)

			actual, metrics, err := triggerChecker.fetch()
			expected := metricSource.MakeTriggerMetricsData(metricData, addMetricData)

			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, expected)
			c.So(metrics, ShouldResemble, []string{metric, addMetric})
		})

		Convey("Two targets with many metrics in additional target", t, func(c C) {
			metricData := []*metricSource.MetricData{metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, from)}
			addMetricData := []*metricSource.MetricData{
				metricSource.MakeMetricData(addMetric, []float64{0, 1, 2, 3, 4}, retention, from),
				metricSource.MakeMetricData(addMetric2, []float64{0, 1, 2, 3, 4}, retention, from),
			}

			source.EXPECT().Fetch(pattern, from, until, false).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return(metricData)
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)

			source.EXPECT().Fetch(addPattern, from, until, false).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return(addMetricData)
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{addMetric, addMetric2}, nil)

			actual, metrics, err := triggerChecker.fetch()
			c.So(err, ShouldBeError)
			c.So(err, ShouldResemble, ErrWrongTriggerTargets([]int{2}))
			c.So(err.Error(), ShouldResemble, "Target t2 has more than one timeseries")
			c.So(actual, ShouldBeNil)
			c.So(metrics, ShouldBeNil)
		})

		Convey("Four targets with many metrics in additional targets", t, func(c C) {
			triggerChecker.trigger.Targets = []string{pattern, addPattern, pattern2, oneMorePattern}
			triggerChecker.trigger.Patterns = []string{pattern, addPattern, pattern2, oneMorePattern}

			metricData := []*metricSource.MetricData{metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, from)}
			add1MetricData := []*metricSource.MetricData{
				metricSource.MakeMetricData(addMetric, []float64{0, 1, 2, 3, 4}, retention, from),
				metricSource.MakeMetricData(addMetric2, []float64{0, 1, 2, 3, 4}, retention, from),
			}
			add2MetricData := []*metricSource.MetricData{metricSource.MakeMetricData(metric2, []float64{0, 1, 2, 3, 4}, retention, from)}
			oneMoreMetricData := []*metricSource.MetricData{
				metricSource.MakeMetricData(oneMoreMetric1, []float64{0, 1, 2, 3, 4}, retention, from),
				metricSource.MakeMetricData(oneMoreMetric2, []float64{0, 1, 2, 3, 4}, retention, from),
			}

			source.EXPECT().Fetch(pattern, from, until, false).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return(metricData)
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)

			source.EXPECT().Fetch(addPattern, from, until, false).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return(add1MetricData)
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{addMetric, addMetric2}, nil)

			source.EXPECT().Fetch(pattern2, from, until, false).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return(add2MetricData)
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric2}, nil)

			source.EXPECT().Fetch(oneMorePattern, from, until, false).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return(oneMoreMetricData)
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{oneMoreMetric1, oneMoreMetric2}, nil)

			actual, metrics, err := triggerChecker.fetch()
			c.So(err, ShouldBeError)
			c.So(err, ShouldResemble, ErrWrongTriggerTargets([]int{2, 4}))
			c.So(err.Error(), ShouldResemble, "Targets t2, t4 has more than one timeseries")
			c.So(actual, ShouldBeNil)
			c.So(metrics, ShouldBeNil)
		})
	})
}

func TestGetExpressionValues(t *testing.T) {
	Convey("Has only main timeSeries", t, func(c C) {
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
		c.So(noEmptyValues, ShouldBeTrue)
		c.So(values, ShouldResemble, expectedExpressionValues)

		values, noEmptyValues = getExpressionValues(tts, metricData, 67)
		c.So(noEmptyValues, ShouldBeFalse)
		c.So(values, ShouldResemble, expectedExpressionValues)

		values, noEmptyValues = getExpressionValues(tts, metricData, 11)
		c.So(noEmptyValues, ShouldBeFalse)
		c.So(values, ShouldResemble, expectedExpressionValues)

		values, noEmptyValues = getExpressionValues(tts, metricData, 44)
		c.So(noEmptyValues, ShouldBeFalse)
		c.So(values, ShouldResemble, expectedExpressionValues)

		expectedExpressionValues.MainTargetValue = 3
		values, noEmptyValues = getExpressionValues(tts, metricData, 53)
		c.So(noEmptyValues, ShouldBeTrue)
		c.So(values, ShouldResemble, expectedExpressionValues)
	})

	Convey("Has additional series", t, func(c C) {
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
		c.So(noEmptyValues, ShouldBeFalse)
		c.So(values, ShouldResemble, expectedExpressionValues)

		values, noEmptyValues = getExpressionValues(tts, metricData, 42)
		c.So(noEmptyValues, ShouldBeFalse)
		c.So(values, ShouldResemble, expectedExpressionValues)

		values, noEmptyValues = getExpressionValues(tts, metricData, 65)
		c.So(noEmptyValues, ShouldBeFalse)
		c.So(values, ShouldResemble, expectedExpressionValues)

		expectedExpressionValues.MainTargetValue = 3
		values, noEmptyValues = getExpressionValues(tts, metricData, 50)
		c.So(noEmptyValues, ShouldBeFalse)
		c.So(values, ShouldResemble, expectedExpressionValues)

		expectedExpressionValues.MainTargetValue = 0
		expectedExpressionValues.AdditionalTargetsValues["t2"] = 4
		values, noEmptyValues = getExpressionValues(tts, metricData, 17)
		c.So(noEmptyValues, ShouldBeTrue)
		c.So(values, ShouldResemble, expectedExpressionValues)
	})
}
