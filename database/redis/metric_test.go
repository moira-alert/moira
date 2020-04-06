package redis

import (
	"testing"
	"time"

	"github.com/op/go-logging"
	"github.com/patrickmn/go-cache"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
)

func TestMetricsStoring(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger, config, testSource)
	dataBase.flush()
	metric1 := "my.test.super.metric"
	metric2 := "my.test.super.metric2"
	pattern := "my.test.*.metric*"
	Convey("GetPatterns works only if you add new trigger with this pattern", t, func() {
		trigger := moira.Trigger{
			ID:       "id",
			Patterns: []string{pattern},
		}
		actual, err := dataBase.GetPatterns()
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		//But you still can add new metrics by this pattern
		err = dataBase.AddPatternMetric(pattern, metric1)
		So(err, ShouldBeNil)

		actualMetric, err := dataBase.GetPatternMetrics(pattern)
		So(err, ShouldBeNil)
		So(actualMetric, ShouldHaveLength, 1)

		err = dataBase.AddPatternMetric(pattern, metric2)
		So(err, ShouldBeNil)

		actualMetric, err = dataBase.GetPatternMetrics(pattern)
		So(err, ShouldBeNil)
		So(actualMetric, ShouldHaveLength, 2)

		//And nothing to remove
		err = dataBase.RemovePattern(pattern)
		So(err, ShouldBeNil)

		actual, err = dataBase.GetPatterns()
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		//Now save trigger with this pattern
		dataBase.SaveTrigger(trigger.ID, &trigger)

		actual, err = dataBase.GetPatterns()
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, trigger.Patterns)

		//And you still can get metrics by this pattern
		actualMetric, err = dataBase.GetPatternMetrics(pattern)
		So(err, ShouldBeNil)
		So(actualMetric, ShouldHaveLength, 2)

		Convey("You can remove pattern and metric separately", func() {
			err = dataBase.RemovePattern(pattern)
			So(err, ShouldBeNil)

			//But you still can get metrics by this pattern
			actualMetric, err = dataBase.GetPatternMetrics(pattern)
			So(err, ShouldBeNil)
			So(actualMetric, ShouldHaveLength, 2)

			err = dataBase.RemovePatternsMetrics([]string{pattern})
			So(err, ShouldBeNil)
		})

		Convey("You can remove remove pattern with metrics in one request", func() {
			err = dataBase.RemovePatternWithMetrics(pattern)
			So(err, ShouldBeNil)
		})

		//Now it have not patterns and metrics for this
		actual, err = dataBase.GetPatterns()
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		//And you still can get metrics by this pattern
		actualMetric, err = dataBase.GetPatternMetrics(pattern)
		So(err, ShouldBeNil)
		So(actualMetric, ShouldBeEmpty)
	})

	Convey("Metrics values and retentions manipulation", t, func() {
		val1 := &moira.MatchedMetric{
			Patterns:           []string{pattern},
			Metric:             metric1,
			Retention:          10,
			RetentionTimestamp: 10,
			Timestamp:          15,
			Value:              1,
		}
		val2 := &moira.MatchedMetric{
			Patterns:           []string{pattern},
			Metric:             metric1,
			Retention:          10,
			RetentionTimestamp: 20,
			Timestamp:          22,
			Value:              2,
		}
		val3 := &moira.MatchedMetric{
			Patterns:           []string{pattern},
			Metric:             metric1,
			Retention:          60,
			RetentionTimestamp: 60,
			Timestamp:          66,
			Value:              3,
		}

		actualRet, err := dataBase.GetMetricRetention(metric1)
		So(err, ShouldBeNil)
		So(actualRet, ShouldEqual, 60)

		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: val1})
		So(err, ShouldBeNil)

		actualRet, err = dataBase.GetMetricRetention(metric1)
		So(err, ShouldBeNil)
		So(actualRet, ShouldEqual, 10)

		actualValues, err := dataBase.GetMetricsValues([]string{metric1}, 0, 9)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {}})

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 0, 10)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {&moira.MetricValue{Timestamp: 15, RetentionTimestamp: 10, Value: 1}}})

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 10, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {&moira.MetricValue{Timestamp: 15, RetentionTimestamp: 10, Value: 1}}})

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 11, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {}})

		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: val2})
		So(err, ShouldBeNil)

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 0, 9)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {}})

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 0, 10)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {&moira.MetricValue{Timestamp: 15, RetentionTimestamp: 10, Value: 1}}})

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 0, 19)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {&moira.MetricValue{Timestamp: 15, RetentionTimestamp: 10, Value: 1}}})

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 10, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {&moira.MetricValue{Timestamp: 15, RetentionTimestamp: 10, Value: 1}, &moira.MetricValue{Timestamp: 22, RetentionTimestamp: 20, Value: 2}}})

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 11, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {&moira.MetricValue{Timestamp: 22, RetentionTimestamp: 20, Value: 2}}})

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 21, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {}})

		//Save metric with changed retention
		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: val3})
		So(err, ShouldBeNil)

		//But retention still old, because cache
		actualRet, err = dataBase.GetMetricRetention(metric1)
		So(err, ShouldBeNil)
		So(actualRet, ShouldEqual, 10)
	})
}

func TestRemoveMetricValues(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger, config, testSource)
	dataBase.metricsCache = cache.New(time.Second*2, time.Minute*60)
	dataBase.flush()
	defer dataBase.flush()
	metric1 := "my.test.super.metric"
	pattern := "my.test.*.metric*"
	met1 := &moira.MatchedMetric{
		Patterns:           []string{pattern},
		Metric:             metric1,
		Retention:          10,
		RetentionTimestamp: 10,
		Timestamp:          15,
		Value:              1,
	}
	met2 := &moira.MatchedMetric{
		Patterns:           []string{pattern},
		Metric:             metric1,
		Retention:          10,
		RetentionTimestamp: 20,
		Timestamp:          24,
		Value:              2,
	}
	met3 := &moira.MatchedMetric{
		Patterns:           []string{pattern},
		Metric:             metric1,
		Retention:          10,
		RetentionTimestamp: 30,
		Timestamp:          34,
		Value:              3,
	}
	met4 := &moira.MatchedMetric{
		Patterns:           []string{pattern},
		Metric:             metric1,
		Retention:          10,
		RetentionTimestamp: 40,
		Timestamp:          46,
		Value:              4,
	}

	Convey("Test", t, func() {
		err := dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: met1})
		So(err, ShouldBeNil) //Save metric with changed retention
		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: met2})
		So(err, ShouldBeNil) //Save metric with changed retention
		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: met3})
		So(err, ShouldBeNil) //Save metric with changed retention
		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: met4})
		So(err, ShouldBeNil)

		actualValues, err := dataBase.GetMetricsValues([]string{metric1}, 1, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
			metric1: {
				&moira.MetricValue{Timestamp: 15, RetentionTimestamp: 10, Value: 1},
				&moira.MetricValue{Timestamp: 24, RetentionTimestamp: 20, Value: 2},
				&moira.MetricValue{Timestamp: 34, RetentionTimestamp: 30, Value: 3},
				&moira.MetricValue{Timestamp: 46, RetentionTimestamp: 40, Value: 4},
			},
		})

		err = dataBase.RemoveMetricValues(metric1, 11)
		So(err, ShouldBeNil)

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 1, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
			metric1: {
				&moira.MetricValue{Timestamp: 24, RetentionTimestamp: 20, Value: 2},
				&moira.MetricValue{Timestamp: 34, RetentionTimestamp: 30, Value: 3},
				&moira.MetricValue{Timestamp: 46, RetentionTimestamp: 40, Value: 4},
			},
		})

		err = dataBase.RemoveMetricValues(metric1, 22)
		So(err, ShouldBeNil)

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 1, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
			metric1: {
				&moira.MetricValue{Timestamp: 24, RetentionTimestamp: 20, Value: 2},
				&moira.MetricValue{Timestamp: 34, RetentionTimestamp: 30, Value: 3},
				&moira.MetricValue{Timestamp: 46, RetentionTimestamp: 40, Value: 4},
			},
		})

		err = dataBase.RemoveMetricsValues([]string{metric1}, 22)
		So(err, ShouldBeNil)

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 1, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
			metric1: {
				&moira.MetricValue{Timestamp: 24, RetentionTimestamp: 20, Value: 2},
				&moira.MetricValue{Timestamp: 34, RetentionTimestamp: 30, Value: 3},
				&moira.MetricValue{Timestamp: 46, RetentionTimestamp: 40, Value: 4},
			},
		})

		time.Sleep(time.Second * 2)

		err = dataBase.RemoveMetricsValues([]string{metric1}, 22)
		So(err, ShouldBeNil)

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 1, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
			metric1: {
				&moira.MetricValue{Timestamp: 34, RetentionTimestamp: 30, Value: 3},
				&moira.MetricValue{Timestamp: 46, RetentionTimestamp: 40, Value: 4},
			},
		})

		time.Sleep(time.Second * 2)

		err = dataBase.RemoveMetricValues(metric1, 30)
		So(err, ShouldBeNil)

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 1, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
			metric1: {
				&moira.MetricValue{Timestamp: 46, RetentionTimestamp: 40, Value: 4},
			},
		})

		time.Sleep(time.Second * 2)

		err = dataBase.RemoveMetricValues(metric1, 39)
		So(err, ShouldBeNil)

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 1, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
			metric1: {
				&moira.MetricValue{Timestamp: 46, RetentionTimestamp: 40, Value: 4},
			},
		})

		time.Sleep(time.Second * 2)

		err = dataBase.RemoveMetricValues(metric1, 49)
		So(err, ShouldBeNil)

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 1, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {}})
	})
}

func TestMetricSubscription(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger, config, testSource)
	dataBase.flush()
	defer dataBase.flush()
	metric1 := "my.test.super.metric"
	metric2 := "my.test.super.metric2"
	pattern := "my.test.*.metric*"
	Convey("Subscription manipulation", t, func() {
		var tomb1 tomb.Tomb
		ch, err := dataBase.SubscribeMetricEvents(&tomb1)
		So(err, ShouldBeNil)
		So(ch, ShouldNotBeNil)

		met1 := &moira.MatchedMetric{
			Patterns:           []string{pattern},
			Metric:             metric1,
			Retention:          10,
			RetentionTimestamp: 10,
			Timestamp:          15,
			Value:              1,
		}

		met2 := &moira.MatchedMetric{
			Patterns:           []string{pattern},
			Metric:             metric2,
			Retention:          20,
			RetentionTimestamp: 20,
			Timestamp:          25,
			Value:              2,
		}
		numberOfChecks := 0

		tomb1.Go(func() error {
			for {
				metricEvent, ok := <-ch
				if !ok {
					numberOfChecks++
					logger.Info("Channel closed, end test")
					return nil
				}
				if metricEvent.Metric == metric1 {
					Convey("Test", t, func() {
						numberOfChecks++
						So(metricEvent, ShouldResemble, &moira.MetricEvent{Pattern: pattern, Metric: metric1})
					})
				}
				if metricEvent.Metric == metric2 {
					Convey("Test", t, func() {
						numberOfChecks++
						So(metricEvent, ShouldResemble, &moira.MetricEvent{Pattern: pattern, Metric: metric2})
					})
				}
			}
		})

		dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: met1})
		dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric2: met2})
		tomb1.Kill(nil)
		tomb1.Wait()

		So(numberOfChecks, ShouldEqual, 3)
	})
}

func TestMetricsStoringErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger, emptyConfig, testSource)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func() {
		actual, err := dataBase.GetPatterns()
		So(actual, ShouldBeEmpty)
		So(err, ShouldNotBeNil)

		actual1, err := dataBase.GetMetricsValues([]string{"123"}, 0, 1)
		So(actual1, ShouldBeEmpty)
		So(err, ShouldNotBeNil)

		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{"metric1": {Value: 1, RetentionTimestamp: 1, Timestamp: 1, Retention: 60, Patterns: []string{"12"}, Metric: "123"}})
		So(err, ShouldNotBeNil)

		actual2, err := dataBase.GetMetricRetention("123")
		So(actual2, ShouldEqual, 0)
		So(err, ShouldNotBeNil)

		err = dataBase.AddPatternMetric("123", "123234")
		So(err, ShouldNotBeNil)

		actual, err = dataBase.GetPatternMetrics("123")
		So(actual, ShouldBeEmpty)
		So(err, ShouldNotBeNil)

		err = dataBase.RemovePattern("123")
		So(err, ShouldNotBeNil)

		err = dataBase.RemovePatternsMetrics([]string{"123"})
		So(err, ShouldNotBeNil)

		err = dataBase.RemovePatternWithMetrics("123")
		So(err, ShouldNotBeNil)

		err = dataBase.RemoveMetricValues("123", 1)
		So(err, ShouldNotBeNil)

		var tomb1 tomb.Tomb
		ch, err := dataBase.SubscribeMetricEvents(&tomb1)
		So(err, ShouldNotBeNil)
		So(ch, ShouldBeNil)
	})
}
