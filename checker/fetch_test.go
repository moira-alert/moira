package checker

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFetchTriggerMetrics(t *testing.T) {

	Convey("Test fetch trigger metrics", t, func() {
		mockCtrl := gomock.NewController(t)
		source := mock_metric_source.NewMockMetricSource(mockCtrl)
		fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
		database := mock_moira_alert.NewMockDatabase(mockCtrl)
		defer mockCtrl.Finish()

		var from int64 = 17
		var until int64 = 67
		pattern := "super.puper.pattern"
		var metricsTTL int64 = 3600

		triggerChecker := &TriggerChecker{
			source:   source,
			from:     from,
			until:    until,
			database: database,
			config:   &Config{},
			trigger: &moira.Trigger{
				Targets:  []string{pattern},
				Patterns: []string{pattern},
			},
			lastCheck: &moira.CheckData{
				Metrics: map[string]moira.MetricState{},
			},
		}

		Convey("no metrics in last check", func() {
			Convey("fetch returns wildcard", func() {
				gomock.InOrder(
					source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil),
					fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{{Name: pattern, Wildcard: true}}),
					fetchResult.EXPECT().GetPatternMetrics().Return([]string{pattern}, nil),
					database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL),
					database.EXPECT().RemoveMetricsValues([]string{pattern}, until-metricsTTL).Return(nil),
				)
				actual, err := triggerChecker.fetchTriggerMetrics()
				So(err, ShouldResemble, ErrTriggerHasOnlyWildcards{})
				So(actual, ShouldResemble, map[string][]metricSource.MetricData{"t1": []metricSource.MetricData{{Name: pattern, Wildcard: true}}})
			})

			Convey("fetch returns no metrics", func() {
				source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil)
				fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{})
				fetchResult.EXPECT().GetPatternMetrics().Return([]string{}, nil)

				actual, err := triggerChecker.fetchTriggerMetrics()
				So(err, ShouldResemble, ErrTargetHasNoMetrics{targetIndex: 1})
				So(actual, ShouldBeNil)
			})
		})

		Convey("has metrics in last check", func() {
			triggerChecker.lastCheck.Metrics["metric"] = moira.MetricState{}
			Convey("fetch returns wildcard", func() {
				gomock.InOrder(
					source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil),
					fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{{Name: pattern, Wildcard: true}}),
					fetchResult.EXPECT().GetPatternMetrics().Return([]string{pattern}, nil),
					database.EXPECT().GetMetricsTTLSeconds().Return(metricsTTL),
					database.EXPECT().RemoveMetricsValues([]string{pattern}, until-metricsTTL).Return(nil),
				)

				actual, err := triggerChecker.fetchTriggerMetrics()
				So(err, ShouldBeEmpty)
				So(actual, ShouldResemble, map[string][]metricSource.MetricData{"t1": []metricSource.MetricData{{Name: pattern, Wildcard: true}}})
			})

			Convey("fetch returns no metrics", func() {
				source.EXPECT().Fetch(pattern, triggerChecker.from, triggerChecker.until, true).Return(fetchResult, nil)
				fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{})
				fetchResult.EXPECT().GetPatternMetrics().Return([]string{}, nil)

				actual, err := triggerChecker.fetchTriggerMetrics()
				So(err, ShouldResemble, ErrTargetHasNoMetrics{targetIndex: 1})
				So(actual, ShouldBeNil)
			})
		})
	})
}

func TestFetch(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	source := mock_metric_source.NewMockMetricSource(mockCtrl)
	fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
	defer mockCtrl.Finish()

	pattern := "super.puper.pattern"
	metric := "super.puper.metric"

	addPattern := "additional.pattern"
	addMetric := "additional.metric"
	addMetric2 := "additional.metric2"

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

	Convey("Error test", t, func() {
		metricErr := fmt.Errorf("ooops, metric error")
		source.EXPECT().Fetch(pattern, from, until, true).Return(nil, metricErr)
		actual, metrics, err := triggerChecker.fetch()
		So(actual, ShouldBeNil)
		So(metrics, ShouldBeNil)
		So(err, ShouldBeError)
		So(err, ShouldResemble, metricErr)
	})

	Convey("Test no metrics in target", t, func() {
		metricData := metricSource.MetricData{
			Name:      pattern,
			StartTime: from,
			StopTime:  until,
			StepTime:  60,
			Values:    []float64{},
			Wildcard:  true,
		}

		source.EXPECT().Fetch(pattern, from, until, true).Return(fetchResult, nil)
		fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{metricData})
		fetchResult.EXPECT().GetPatternMetrics().Return([]string{}, nil)
		actual, metrics, err := triggerChecker.fetch()
		So(actual, ShouldBeNil)
		So(metrics, ShouldBeEmpty)
		So(err, ShouldResemble, ErrTargetHasNoMetrics{targetIndex: 1})
	})

	Convey("Test has metrics", t, func() {
		Convey("Only one target", func() {
			source.EXPECT().Fetch(pattern, from, until, true).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, from)})
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)
			actual, metrics, err := triggerChecker.fetch()
			metricData := metricSource.MetricData{
				Name:      metric,
				StartTime: from,
				StopTime:  until,
				StepTime:  retention,
				Values:    []float64{0, 1, 2, 3, 4},
			}
			expected := map[string][]metricSource.MetricData{"t1": []metricSource.MetricData{metricData}}
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
			So(metrics, ShouldResemble, []string{metric})
		})

		Convey("Two targets", func() {
			triggerChecker.trigger.Targets = []string{pattern, addPattern}
			triggerChecker.trigger.Patterns = []string{pattern, addPattern}

			metricData := []metricSource.MetricData{*metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, from)}
			addMetricData := []metricSource.MetricData{*metricSource.MakeMetricData(addMetric, []float64{0, 1, 2, 3, 4}, retention, from)}

			source.EXPECT().Fetch(pattern, from, until, false).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return(metricData)
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)

			source.EXPECT().Fetch(addPattern, from, until, false).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return(addMetricData)
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{addMetric}, nil)

			actual, metrics, err := triggerChecker.fetch()
			expected := map[string][]metricSource.MetricData{"t1": metricData, "t2": addMetricData}

			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
			So(metrics, ShouldResemble, []string{metric, addMetric})
		})

		Convey("Two targets with many metrics in additional target", func() {
			metricData := []metricSource.MetricData{*metricSource.MakeMetricData(metric, []float64{0, 1, 2, 3, 4}, retention, from)}

			addMetricData := []metricSource.MetricData{
				*metricSource.MakeMetricData(addMetric, []float64{0, 1, 2, 3, 4}, retention, from),
				*metricSource.MakeMetricData(addMetric2, []float64{0, 1, 2, 3, 4}, retention, from),
			}

			source.EXPECT().Fetch(pattern, from, until, false).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return(metricData)
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{metric}, nil)

			source.EXPECT().Fetch(addPattern, from, until, false).Return(fetchResult, nil)
			fetchResult.EXPECT().GetMetricsData().Return(addMetricData)
			fetchResult.EXPECT().GetPatternMetrics().Return([]string{addMetric, addMetric2}, nil)

			actual, metrics, err := triggerChecker.fetch()
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, map[string][]metricSource.MetricData{"t1": metricData, "t2": addMetricData})
			So(metrics, ShouldResemble, []string{metric, addMetric, addMetric2})
		})
	})
}
