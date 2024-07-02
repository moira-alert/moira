package redis

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	"github.com/patrickmn/go-cache"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gopkg.in/tomb.v2"
)

const (
	toInf   = "+inf"
	fromInf = "-inf"
)

func TestMetricsStoring(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	metric1 := "my.test.super.metric" //nolint
	metric2 := "my.test.super.metric2"
	pattern := "my.test.*.metric*" //nolint
	Convey("GetPatterns works only if you add new trigger with this pattern", t, func() {
		trigger := moira.Trigger{
			ID:       "id",
			Patterns: []string{pattern},
		}
		actual, err := dataBase.GetPatterns()
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		// But you still can add new metrics by this pattern
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

		// And nothing to remove
		err = dataBase.RemovePattern(pattern)
		So(err, ShouldBeNil)

		actual, err = dataBase.GetPatterns()
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		// Now save trigger with this pattern
		dataBase.SaveTrigger(trigger.ID, &trigger) //nolint

		actual, err = dataBase.GetPatterns()
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, trigger.Patterns)

		// And you still can get metrics by this pattern
		actualMetric, err = dataBase.GetPatternMetrics(pattern)
		So(err, ShouldBeNil)
		So(actualMetric, ShouldHaveLength, 2)

		Convey("You can remove pattern and metric separately", func() {
			err = dataBase.RemovePattern(pattern)
			So(err, ShouldBeNil)

			// But you still can get metrics by this pattern
			actualMetric, err = dataBase.GetPatternMetrics(pattern)
			So(err, ShouldBeNil)
			So(actualMetric, ShouldHaveLength, 2)

			err = dataBase.RemovePatternsMetrics([]string{pattern})
			So(err, ShouldBeNil)
		})

		Convey("You can remove pattern with metrics in one request", func() {
			err = dataBase.RemovePatternWithMetrics(pattern)
			So(err, ShouldBeNil)
		})

		// Now it have not patterns and metrics for this
		actual, err = dataBase.GetPatterns()
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		// And you still can get metrics by this pattern
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

		// Save metric with changed retention
		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: val3})
		So(err, ShouldBeNil)

		// But retention still old, because cache
		actualRet, err = dataBase.GetMetricRetention(metric1)
		So(err, ShouldBeNil)
		So(actualRet, ShouldEqual, 10)
	})
}

func TestRemoveMetricRetention(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "warn", "test", true)
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	Convey("Given metric", t, func() {
		const metric = "my.test.super.metric"

		tsOlder := time.Now().UTC().Add(-80 * time.Second).Unix()
		tsNow := time.Now().UTC().Unix()
		metric1Value := &moira.MatchedMetric{
			Metric:             metric,
			Retention:          10,
			RetentionTimestamp: tsOlder,
			Timestamp:          tsOlder,
		}

		err := dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric1Value})
		So(err, ShouldBeNil)

		actualValues, err := dataBase.GetMetricsValues([]string{metric}, 0, tsNow)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
			metric: {
				&moira.MetricValue{
					RetentionTimestamp: tsOlder,
					Timestamp:          tsOlder,
				},
			},
		})

		Convey("When remove metric retention", func() {
			client := *dataBase.client

			err = dataBase.RemoveMetricRetention(metric)
			So(err, ShouldBeNil)

			Convey("metric retention key shouldn't be in database", func() {
				isMetricRetentionExists := client.Exists(dataBase.context, metricRetentionKey(metric)).Val() == 1
				So(isMetricRetentionExists, ShouldBeFalse)
			})
		})
	})
}

func TestRemoveMetricValues(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.metricsCache = cache.New(time.Second*2, time.Minute*60)
	dataBase.Flush()
	defer dataBase.Flush()

	Convey("Test that old metrics will be deleted", t, func() {
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

		from := fromInf

		err := dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: met1})
		So(err, ShouldBeNil) // Save metric with changed retention
		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: met2})
		So(err, ShouldBeNil) // Save metric with changed retention
		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: met3})
		So(err, ShouldBeNil) // Save metric with changed retention
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

		var toTs int64 = 11
		to := strconv.FormatInt(toTs, 10)
		deletedCount, err := dataBase.RemoveMetricValues(metric1, from, to)
		So(err, ShouldBeNil)
		So(deletedCount, ShouldResemble, int64(1))

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 1, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
			metric1: {
				&moira.MetricValue{Timestamp: 24, RetentionTimestamp: 20, Value: 2},
				&moira.MetricValue{Timestamp: 34, RetentionTimestamp: 30, Value: 3},
				&moira.MetricValue{Timestamp: 46, RetentionTimestamp: 40, Value: 4},
			},
		})

		toTs = 22
		to = strconv.FormatInt(toTs, 10)
		deletedCount, err = dataBase.RemoveMetricValues(metric1, from, to)
		So(err, ShouldBeNil)
		So(deletedCount, ShouldResemble, int64(0))

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

		dataBase.metricsCache.Flush()

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

		dataBase.metricsCache.Flush()

		toTs = 30
		to = strconv.FormatInt(toTs, 10)
		deletedCount, err = dataBase.RemoveMetricValues(metric1, from, to)
		So(err, ShouldBeNil)
		So(deletedCount, ShouldResemble, int64(1))

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 1, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
			metric1: {
				&moira.MetricValue{Timestamp: 46, RetentionTimestamp: 40, Value: 4},
			},
		})

		dataBase.metricsCache.Flush()

		toTs = 39
		to = strconv.FormatInt(toTs, 10)
		deletedCount, err = dataBase.RemoveMetricValues(metric1, from, to)
		So(err, ShouldBeNil)
		So(deletedCount, ShouldResemble, int64(0))

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 1, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
			metric1: {
				&moira.MetricValue{Timestamp: 46, RetentionTimestamp: 40, Value: 4},
			},
		})

		dataBase.metricsCache.Flush()

		toTs = 49
		to = strconv.FormatInt(toTs, 10)
		deletedCount, err = dataBase.RemoveMetricValues(metric1, from, to)
		So(err, ShouldBeNil)
		So(deletedCount, ShouldResemble, int64(1))

		actualValues, err = dataBase.GetMetricsValues([]string{metric1}, 1, 99)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric1: {}})
	})

	Convey("Test remove metric values", t, func() {
		metric := "metric1"

		metric1 := &moira.MatchedMetric{
			Metric:             metric,
			RetentionTimestamp: 10,
			Timestamp:          10,
			Retention:          10,
			Value:              1,
		}

		metric2 := &moira.MatchedMetric{
			Metric:             metric,
			RetentionTimestamp: 20,
			Timestamp:          20,
			Retention:          10,
			Value:              2,
		}

		metric3 := &moira.MatchedMetric{
			Metric:             metric,
			RetentionTimestamp: 30,
			Timestamp:          30,
			Retention:          10,
			Value:              3,
		}

		metric4 := &moira.MatchedMetric{
			Metric:             metric,
			RetentionTimestamp: 40,
			Timestamp:          40,
			Retention:          10,
			Value:              4,
		}

		metric5 := &moira.MatchedMetric{
			Metric:             metric,
			RetentionTimestamp: 50,
			Timestamp:          50,
			Retention:          10,
			Value:              5,
		}

		Convey("Deleting metrics over the entire interval (from -inf to +inf)", func() {
			defer func() {
				dataBase.metricsCache.Flush()
			}()

			from := fromInf
			to := toInf

			err := dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric1})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric2})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric3})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric4})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric5})
			So(err, ShouldBeNil)

			dataBase.metricsCache.Flush()

			actualValues, err := dataBase.GetMetricsValues([]string{metric}, 1, 99)
			So(err, ShouldBeNil)
			So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
				metric: {
					&moira.MetricValue{Timestamp: 10, RetentionTimestamp: 10, Value: 1},
					&moira.MetricValue{Timestamp: 20, RetentionTimestamp: 20, Value: 2},
					&moira.MetricValue{Timestamp: 30, RetentionTimestamp: 30, Value: 3},
					&moira.MetricValue{Timestamp: 40, RetentionTimestamp: 40, Value: 4},
					&moira.MetricValue{Timestamp: 50, RetentionTimestamp: 50, Value: 5},
				},
			})

			deletedMetrics, err := dataBase.RemoveMetricValues(metric, from, to)
			So(err, ShouldBeNil)
			So(deletedMetrics, ShouldEqual, 5)

			actualValues, err = dataBase.GetMetricsValues([]string{metric}, 1, 99)
			So(err, ShouldBeNil)
			So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{metric: {}})
		})

		Convey("Deletion of metrics on the interval up to the first metric (from -inf to 5)", func() {
			defer func() {
				dataBase.metricsCache.Flush()

				deletedMetrics, err := dataBase.RemoveMetricValues(metric, fromInf, toInf)
				So(err, ShouldBeNil)
				So(deletedMetrics, ShouldEqual, 5)
			}()

			from := fromInf
			to := "5"

			err := dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric1})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric2})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric3})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric4})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric5})
			So(err, ShouldBeNil)

			dataBase.metricsCache.Flush()

			deletedMetrics, err := dataBase.RemoveMetricValues(metric, from, to)
			So(err, ShouldBeNil)
			So(deletedMetrics, ShouldEqual, 0)

			actualValues, err := dataBase.GetMetricsValues([]string{metric}, 1, 99)
			So(err, ShouldBeNil)
			So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
				metric: {
					&moira.MetricValue{Timestamp: 10, RetentionTimestamp: 10, Value: 1},
					&moira.MetricValue{Timestamp: 20, RetentionTimestamp: 20, Value: 2},
					&moira.MetricValue{Timestamp: 30, RetentionTimestamp: 30, Value: 3},
					&moira.MetricValue{Timestamp: 40, RetentionTimestamp: 40, Value: 4},
					&moira.MetricValue{Timestamp: 50, RetentionTimestamp: 50, Value: 5},
				},
			})
		})

		Convey("Deleting metrics on the interval after the last metric (from 60 to +inf)", func() {
			defer func() {
				dataBase.metricsCache.Flush()

				deletedMetrics, err := dataBase.RemoveMetricValues(metric, fromInf, toInf)
				So(err, ShouldBeNil)
				So(deletedMetrics, ShouldEqual, 5)
			}()

			from := "60"
			to := toInf

			err := dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric1})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric2})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric3})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric4})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric5})
			So(err, ShouldBeNil)

			dataBase.metricsCache.Flush()

			deletedMetrics, err := dataBase.RemoveMetricValues(metric, from, to)
			So(err, ShouldBeNil)
			So(deletedMetrics, ShouldEqual, 0)

			actualValues, err := dataBase.GetMetricsValues([]string{metric}, 1, 99)
			So(err, ShouldBeNil)
			So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
				metric: {
					&moira.MetricValue{Timestamp: 10, RetentionTimestamp: 10, Value: 1},
					&moira.MetricValue{Timestamp: 20, RetentionTimestamp: 20, Value: 2},
					&moira.MetricValue{Timestamp: 30, RetentionTimestamp: 30, Value: 3},
					&moira.MetricValue{Timestamp: 40, RetentionTimestamp: 40, Value: 4},
					&moira.MetricValue{Timestamp: 50, RetentionTimestamp: 50, Value: 5},
				},
			})
		})

		Convey("Deleting metrics inside the metric interval (from 20 to 40)", func() {
			defer func() {
				dataBase.metricsCache.Flush()

				deletedMetrics, err := dataBase.RemoveMetricValues(metric, fromInf, toInf)
				So(err, ShouldBeNil)
				So(deletedMetrics, ShouldEqual, 2)
			}()

			from := "20"
			to := "40"

			err := dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric1})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric2})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric3})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric4})
			So(err, ShouldBeNil)
			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric: metric5})
			So(err, ShouldBeNil)

			dataBase.metricsCache.Flush()

			deletedMetrics, err := dataBase.RemoveMetricValues(metric, from, to)
			So(err, ShouldBeNil)
			So(deletedMetrics, ShouldEqual, 3)

			actualValues, err := dataBase.GetMetricsValues([]string{metric}, 1, 99)
			So(err, ShouldBeNil)
			So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
				metric: {
					&moira.MetricValue{Timestamp: 10, RetentionTimestamp: 10, Value: 1},
					&moira.MetricValue{Timestamp: 50, RetentionTimestamp: 50, Value: 5},
				},
			})
		})
	})
}

func TestMetricSubscription(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)

	dataBase.Flush()
	defer dataBase.Flush()
	metric1 := "my.test.super.metric"
	metric2 := "my.test.super.metric2"
	pattern := "my.test.*.metric*"
	Convey("Subscription manipulation", t, func() {
		var tomb1 tomb.Tomb
		ch, err := dataBase.SubscribeMetricEvents(&tomb1,
			&moira.SubscribeMetricEventsParams{BatchSize: 100, Delay: time.Duration(0)})
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
					logger.Info().Msg("Channel closed, end test")
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

		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: met1})
		So(err, ShouldBeNil)
		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric2: met2})
		So(err, ShouldBeNil)
		time.Sleep(time.Second * 6)
		tomb1.Kill(nil)
		err = tomb1.Wait()
		So(err, ShouldBeNil)

		So(numberOfChecks, ShouldEqual, 3)
	})
}

func TestMetricsStoringErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabaseWithIncorrectConfig(logger)
	dataBase.Flush()
	defer dataBase.Flush()
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

		from := fromInf
		var toTs int64 = 1
		to := strconv.FormatInt(toTs, 10)
		deletedCount, err := dataBase.RemoveMetricValues("123", from, to)
		So(err, ShouldNotBeNil)
		So(deletedCount, ShouldResemble, int64(0))

		var tomb1 tomb.Tomb
		ch, err := dataBase.SubscribeMetricEvents(&tomb1,
			&moira.SubscribeMetricEventsParams{BatchSize: 100, Delay: time.Duration(0)})
		So(err, ShouldNotBeNil)
		So(ch, ShouldBeNil)
	})
}

func TestCleanupOutdatedMetrics(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "warn", "test", true)
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	Convey("Given 2 metrics with 2 values older then 1 minute and 2 values younger then 1 minute", t, func() {
		const (
			metric1 = "my.test.super.metric"
			metric2 = "my.test.super.metric2"
			pattern = "my.test.*.metric*"
		)

		tsOlder1 := time.Now().UTC().Add(-80 * time.Second).Unix()
		tsOlder2 := time.Now().UTC().Add(-70 * time.Second).Unix()
		tsYounger1 := time.Now().UTC().Add(-50 * time.Second).Unix()
		tsYounger2 := time.Now().UTC().Add(-40 * time.Second).Unix()
		tsNow := time.Now().UTC().Unix()
		metric1Value1 := &moira.MatchedMetric{
			Patterns:           []string{pattern},
			Metric:             metric1,
			Retention:          10,
			RetentionTimestamp: tsOlder1,
			Timestamp:          tsOlder1 + 5,
			Value:              1,
		}
		metric1Value2 := &moira.MatchedMetric{
			Patterns:           []string{pattern},
			Metric:             metric1,
			Retention:          10,
			RetentionTimestamp: tsOlder2,
			Timestamp:          tsOlder2 + 5,
			Value:              2,
		}
		metric1Value3 := &moira.MatchedMetric{
			Patterns:           []string{pattern},
			Metric:             metric1,
			Retention:          10,
			RetentionTimestamp: tsYounger1,
			Timestamp:          tsYounger1 + 5,
			Value:              3,
		}
		metric1Value4 := &moira.MatchedMetric{
			Patterns:           []string{pattern},
			Metric:             metric1,
			Retention:          10,
			RetentionTimestamp: tsYounger2,
			Timestamp:          tsYounger2 + 5,
			Value:              4,
		}

		metric2Value1 := &moira.MatchedMetric{
			Patterns:           []string{pattern},
			Metric:             metric2,
			Retention:          10,
			RetentionTimestamp: tsOlder1,
			Timestamp:          tsOlder1 + 5,
			Value:              1,
		}
		metric2Value2 := &moira.MatchedMetric{
			Patterns:           []string{pattern},
			Metric:             metric2,
			Retention:          10,
			RetentionTimestamp: tsOlder2,
			Timestamp:          tsOlder2 + 5,
			Value:              2,
		}
		metric2Value3 := &moira.MatchedMetric{
			Patterns:           []string{pattern},
			Metric:             metric2,
			Retention:          10,
			RetentionTimestamp: tsYounger1,
			Timestamp:          tsYounger1 + 5,
			Value:              3,
		}
		metric2Value4 := &moira.MatchedMetric{
			Patterns:           []string{pattern},
			Metric:             metric2,
			Retention:          10,
			RetentionTimestamp: tsYounger2,
			Timestamp:          tsYounger2 + 5,
			Value:              4,
		}

		err := dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: metric1Value1})
		So(err, ShouldBeNil)
		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: metric1Value2})
		So(err, ShouldBeNil)
		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: metric1Value3})
		So(err, ShouldBeNil)
		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: metric1Value4})
		So(err, ShouldBeNil)

		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric2: metric2Value1})
		So(err, ShouldBeNil)
		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric2: metric2Value2})
		So(err, ShouldBeNil)
		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric2: metric2Value3})
		So(err, ShouldBeNil)
		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric2: metric2Value4})
		So(err, ShouldBeNil)

		actualValues, err := dataBase.GetMetricsValues([]string{metric1, metric2}, 0, tsNow)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
			metric1: {
				&moira.MetricValue{Timestamp: tsOlder1 + 5, RetentionTimestamp: tsOlder1, Value: 1},
				&moira.MetricValue{Timestamp: tsOlder2 + 5, RetentionTimestamp: tsOlder2, Value: 2},
				&moira.MetricValue{Timestamp: tsYounger1 + 5, RetentionTimestamp: tsYounger1, Value: 3},
				&moira.MetricValue{Timestamp: tsYounger2 + 5, RetentionTimestamp: tsYounger2, Value: 4},
			},
			metric2: {
				&moira.MetricValue{Timestamp: tsOlder1 + 5, RetentionTimestamp: tsOlder1, Value: 1},
				&moira.MetricValue{Timestamp: tsOlder2 + 5, RetentionTimestamp: tsOlder2, Value: 2},
				&moira.MetricValue{Timestamp: tsYounger1 + 5, RetentionTimestamp: tsYounger1, Value: 3},
				&moira.MetricValue{Timestamp: tsYounger2 + 5, RetentionTimestamp: tsYounger2, Value: 4},
			},
		})

		Convey("When clean up metrics with wrong value of duration was called (positive number)", func() {
			err = dataBase.CleanUpOutdatedMetrics(time.Hour)
			So(
				err,
				ShouldResemble,
				errors.New("clean up duration value must be less than zero, otherwise all metrics will be removed"),
			)
		})

		Convey("When clean up metrics older then 1 minute was called", func() {
			err = dataBase.CleanUpOutdatedMetrics(-time.Minute)
			So(err, ShouldBeNil)

			Convey("No metrics older then 1 minute should be in database", func() {
				actualValues, err = dataBase.GetMetricsValues([]string{metric1, metric2}, 0, tsNow)
				So(err, ShouldBeNil)
				So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
					metric1: {
						&moira.MetricValue{Timestamp: tsYounger1 + 5, RetentionTimestamp: tsYounger1, Value: 3},
						&moira.MetricValue{Timestamp: tsYounger2 + 5, RetentionTimestamp: tsYounger2, Value: 4},
					},
					metric2: {
						&moira.MetricValue{Timestamp: tsYounger1 + 5, RetentionTimestamp: tsYounger1, Value: 3},
						&moira.MetricValue{Timestamp: tsYounger2 + 5, RetentionTimestamp: tsYounger2, Value: 4},
					},
				})
			})
		})
	})
}

func TestCleanupFutureMetrics(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "warn", "test", true)
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClock := mock_clock.NewMockClock(mockCtrl)

	dataBase.clock = mockClock

	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC)
	testTimeUnix := testTime.Unix()

	retention := 10

	Convey("Test clean up future metrics", t, func() {
		const (
			metric1 = "metric1"
			metric2 = "metric2"
			metric3 = "metric3"
		)

		matchedMetric1 := &moira.MatchedMetric{
			Metric:             metric1,
			Value:              1,
			Timestamp:          testTimeUnix,
			RetentionTimestamp: testTimeUnix,
			Retention:          retention,
		}

		matchedMetric2 := &moira.MatchedMetric{
			Metric:             metric1,
			Value:              2,
			Timestamp:          testTimeUnix + int64(retention),
			RetentionTimestamp: testTimeUnix + int64(retention),
			Retention:          retention,
		}

		matchedMetric3 := &moira.MatchedMetric{
			Metric:             metric2,
			Value:              3,
			Timestamp:          testTimeUnix,
			RetentionTimestamp: testTimeUnix,
			Retention:          retention,
		}

		matchedMetric4 := &moira.MatchedMetric{
			Metric:             metric2,
			Value:              4,
			Timestamp:          testTimeUnix + int64(retention),
			RetentionTimestamp: testTimeUnix + int64(retention),
			Retention:          retention,
		}

		matchedMetric5 := &moira.MatchedMetric{
			Metric:             metric3,
			Value:              5,
			Timestamp:          testTimeUnix,
			RetentionTimestamp: testTimeUnix,
			Retention:          retention,
		}

		Convey("Without future metrics", func() {
			defer func() {
				dataBase.metricsCache.Flush()
			}()

			err := dataBase.SaveMetrics(map[string]*moira.MatchedMetric{
				metric1: matchedMetric1,
				metric2: matchedMetric3,
				metric3: matchedMetric5,
			})
			So(err, ShouldBeNil)

			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{
				metric1: matchedMetric2,
				metric2: matchedMetric4,
			})
			So(err, ShouldBeNil)

			mockClock.EXPECT().Now().Return(testTime).Times(1)

			err = dataBase.CleanUpFutureMetrics(time.Hour)
			So(err, ShouldBeNil)

			actualMetrics, err := dataBase.GetMetricsValues([]string{metric1, metric2, metric3}, testTimeUnix, testTimeUnix+int64(retention))
			So(actualMetrics, ShouldResemble, map[string][]*moira.MetricValue{
				metric1: {
					{
						RetentionTimestamp: testTimeUnix,
						Timestamp:          testTimeUnix,
						Value:              1,
					},
					{
						RetentionTimestamp: testTimeUnix + int64(retention),
						Timestamp:          testTimeUnix + int64(retention),
						Value:              2,
					},
				},
				metric2: {
					{
						RetentionTimestamp: testTimeUnix,
						Timestamp:          testTimeUnix,
						Value:              3,
					},
					{
						RetentionTimestamp: testTimeUnix + int64(retention),
						Timestamp:          testTimeUnix + int64(retention),
						Value:              4,
					},
				},
				metric3: {
					{
						RetentionTimestamp: testTimeUnix,
						Timestamp:          testTimeUnix,
						Value:              5,
					},
				},
			})
			So(err, ShouldBeNil)
		})

		Convey("With future metrics", func() {
			defer func() {
				dataBase.metricsCache.Flush()
			}()

			err := dataBase.SaveMetrics(map[string]*moira.MatchedMetric{
				metric1: matchedMetric1,
				metric2: matchedMetric3,
				metric3: matchedMetric5,
			})
			So(err, ShouldBeNil)

			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{
				metric1: matchedMetric2,
				metric2: matchedMetric4,
			})
			So(err, ShouldBeNil)

			mockClock.EXPECT().Now().Return(testTime).Times(1)

			err = dataBase.CleanUpFutureMetrics(5 * time.Second)
			So(err, ShouldBeNil)

			actualMetrics, err := dataBase.GetMetricsValues([]string{metric1, metric2, metric3}, testTimeUnix, testTimeUnix+int64(retention))
			So(actualMetrics, ShouldResemble, map[string][]*moira.MetricValue{
				metric1: {
					{
						RetentionTimestamp: testTimeUnix,
						Timestamp:          testTimeUnix,
						Value:              1,
					},
				},
				metric2: {
					{
						RetentionTimestamp: testTimeUnix,
						Timestamp:          testTimeUnix,
						Value:              3,
					},
				},
				metric3: {
					{
						RetentionTimestamp: testTimeUnix,
						Timestamp:          testTimeUnix,
						Value:              5,
					},
				},
			})
			So(err, ShouldBeNil)
		})
	})
}

func TestCleanupOutdatedPatternMetrics(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "warn", "test", true)
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	const (
		pattern1 = "pattern1"
		pattern2 = "pattern2"
		pattern3 = "pattern3"

		metric1 = "metric1"
		metric2 = "metric2"
		metric3 = "metric3"
		metric4 = "metric4"
		metric5 = "metric5"
	)

	Convey("Test clean up outdated pattern metrics", t, func() {
		Convey("Without outdated metrics", func() {
			defer func() {
				err := dataBase.RemovePatternWithMetrics(pattern1)
				So(err, ShouldBeNil)

				err = dataBase.RemovePatternWithMetrics(pattern2)
				So(err, ShouldBeNil)

				err = dataBase.RemovePatternWithMetrics(pattern3)
				So(err, ShouldBeNil)
			}()

			matchedMetric1 := &moira.MatchedMetric{
				Patterns: []string{pattern1},
				Metric:   metric1,
			}
			matchedMetric2 := &moira.MatchedMetric{
				Patterns: []string{pattern1},
				Metric:   metric2,
			}
			matchedMetric3 := &moira.MatchedMetric{
				Patterns: []string{pattern2},
				Metric:   metric3,
			}
			matchedMetric4 := &moira.MatchedMetric{
				Patterns: []string{pattern2},
				Metric:   metric4,
			}
			matchedMetric5 := &moira.MatchedMetric{
				Patterns: []string{pattern3},
				Metric:   metric5,
			}

			err := dataBase.addPatterns(pattern1, pattern2, pattern3)
			So(err, ShouldBeNil)

			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{
				metric1: matchedMetric1,
				metric2: matchedMetric2,
				metric3: matchedMetric3,
				metric4: matchedMetric4,
				metric5: matchedMetric5,
			})
			So(err, ShouldBeNil)

			count, err := dataBase.CleanupOutdatedPatternMetrics()
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 0)

			pattern1Metrics, err := dataBase.GetPatternMetrics(pattern1)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, pattern1Metrics, []string{metric1, metric2})

			pattern2Metrics, err := dataBase.GetPatternMetrics(pattern2)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, pattern2Metrics, []string{metric3, metric4})

			pattern3Metrics, err := dataBase.GetPatternMetrics(pattern3)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, pattern3Metrics, []string{metric5})
		})

		Convey("With outdated metrics", func() {
			defer func() {
				err := dataBase.RemovePatternWithMetrics(pattern1)
				So(err, ShouldBeNil)

				err = dataBase.RemovePatternWithMetrics(pattern2)
				So(err, ShouldBeNil)

				err = dataBase.RemovePatternWithMetrics(pattern3)
				So(err, ShouldBeNil)
			}()

			matchedMetric1 := &moira.MatchedMetric{
				Patterns: []string{pattern1},
				Metric:   metric1,
			}
			matchedMetric3 := &moira.MatchedMetric{
				Patterns: []string{pattern2},
				Metric:   metric3,
			}

			err := dataBase.addPatterns(pattern1, pattern2, pattern3)
			So(err, ShouldBeNil)

			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{
				metric1: matchedMetric1,
				metric3: matchedMetric3,
			})
			So(err, ShouldBeNil)

			err = dataBase.AddPatternMetric(pattern1, metric2)
			So(err, ShouldBeNil)

			err = dataBase.AddPatternMetric(pattern2, metric4)
			So(err, ShouldBeNil)

			err = dataBase.AddPatternMetric(pattern3, metric5)
			So(err, ShouldBeNil)

			count, err := dataBase.CleanupOutdatedPatternMetrics()
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 3)

			pattern1Metrics, err := dataBase.GetPatternMetrics(pattern1)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, pattern1Metrics, []string{metric1})

			pattern2Metrics, err := dataBase.GetPatternMetrics(pattern2)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, pattern2Metrics, []string{metric3})

			pattern3Metrics, err := dataBase.GetPatternMetrics(pattern3)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, pattern3Metrics, []string{})
		})
	})
}

func TestGetNonExistentPatternMetrics(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "warn", "test", true)
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	const (
		pattern1 = "pattern1"
		pattern2 = "pattern2"
		pattern3 = "pattern3"

		metric1 = "metric1"
		metric2 = "metric2"
		metric3 = "metric3"
		metric4 = "metric4"
		metric5 = "metric5"
	)

	Convey("Test get non existent pattern metrics", t, func() {
		Convey("Without non existent metrics", func() {
			defer func() {
				err := dataBase.RemovePatternWithMetrics(pattern1)
				So(err, ShouldBeNil)

				err = dataBase.RemovePatternWithMetrics(pattern2)
				So(err, ShouldBeNil)

				err = dataBase.RemovePatternWithMetrics(pattern3)
				So(err, ShouldBeNil)
			}()

			matchedMetric1 := &moira.MatchedMetric{
				Patterns: []string{pattern1},
				Metric:   metric1,
			}
			matchedMetric2 := &moira.MatchedMetric{
				Patterns: []string{pattern1},
				Metric:   metric2,
			}
			matchedMetric3 := &moira.MatchedMetric{
				Patterns: []string{pattern2},
				Metric:   metric3,
			}
			matchedMetric4 := &moira.MatchedMetric{
				Patterns: []string{pattern2},
				Metric:   metric4,
			}
			matchedMetric5 := &moira.MatchedMetric{
				Patterns: []string{pattern3},
				Metric:   metric5,
			}

			err := dataBase.addPatterns(pattern1, pattern2, pattern3)
			So(err, ShouldBeNil)

			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{
				metric1: matchedMetric1,
				metric2: matchedMetric2,
				metric3: matchedMetric3,
				metric4: matchedMetric4,
				metric5: matchedMetric5,
			})
			So(err, ShouldBeNil)

			pattern1Metrics, err := dataBase.getNonExistentPatternMetrics(pattern1)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, pattern1Metrics, []string{})

			pattern2Metrics, err := dataBase.getNonExistentPatternMetrics(pattern2)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, pattern2Metrics, []string{})

			pattern3Metrics, err := dataBase.getNonExistentPatternMetrics(pattern3)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, pattern3Metrics, []string{})
		})

		Convey("With outdated metrics", func() {
			defer func() {
				err := dataBase.RemovePatternWithMetrics(pattern1)
				So(err, ShouldBeNil)

				err = dataBase.RemovePatternWithMetrics(pattern2)
				So(err, ShouldBeNil)

				err = dataBase.RemovePatternWithMetrics(pattern3)
				So(err, ShouldBeNil)
			}()

			matchedMetric1 := &moira.MatchedMetric{
				Patterns: []string{pattern1},
				Metric:   metric1,
			}
			matchedMetric3 := &moira.MatchedMetric{
				Patterns: []string{pattern2},
				Metric:   metric3,
			}

			err := dataBase.addPatterns(pattern1, pattern2, pattern3)
			So(err, ShouldBeNil)

			err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{
				metric1: matchedMetric1,
				metric3: matchedMetric3,
			})
			So(err, ShouldBeNil)

			err = dataBase.AddPatternMetric(pattern1, metric2)
			So(err, ShouldBeNil)

			err = dataBase.AddPatternMetric(pattern2, metric4)
			So(err, ShouldBeNil)

			err = dataBase.AddPatternMetric(pattern3, metric5)
			So(err, ShouldBeNil)

			pattern1Metrics, err := dataBase.getNonExistentPatternMetrics(pattern1)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, pattern1Metrics, []string{metric2})

			pattern2Metrics, err := dataBase.getNonExistentPatternMetrics(pattern2)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, pattern2Metrics, []string{metric4})

			pattern3Metrics, err := dataBase.getNonExistentPatternMetrics(pattern3)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, pattern3Metrics, []string{metric5})
		})
	})
}

func TestCleanupAbandonedRetention(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "warn", "test", true)
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	Convey("Given 2 metrics", t, func() {
		const (
			metric1 = "my.test.super.metric"
			metric2 = "my.test.super.metric2"
		)

		tsOlder := time.Now().UTC().Add(-80 * time.Second).Unix()
		tsNow := time.Now().UTC().Unix()
		metric1Value := &moira.MatchedMetric{
			Metric:             metric1,
			Retention:          10,
			RetentionTimestamp: tsOlder,
			Timestamp:          tsOlder,
		}
		metric2Value := &moira.MatchedMetric{
			Metric:             metric2,
			Retention:          10,
			RetentionTimestamp: tsOlder,
			Timestamp:          tsOlder,
		}

		err := dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: metric1Value})
		So(err, ShouldBeNil)

		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric2: metric2Value})
		So(err, ShouldBeNil)

		actualValues, err := dataBase.GetMetricsValues([]string{metric1, metric2}, 0, tsNow)
		So(err, ShouldBeNil)
		So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
			metric1: {
				&moira.MetricValue{
					RetentionTimestamp: tsOlder,
					Timestamp:          tsOlder,
				},
			},
			metric2: {
				&moira.MetricValue{
					RetentionTimestamp: tsOlder,
					Timestamp:          tsOlder,
				},
			},
		})

		Convey("When clean up retentions was called with existent retention and non-existent metric-data in database", func() {
			client := *dataBase.client

			client.Del(dataBase.context, metricDataKey(metric1))

			actualValues, err = dataBase.GetMetricsValues([]string{metric1, metric2}, 0, tsNow)
			So(err, ShouldBeNil)
			So(actualValues, ShouldResemble, map[string][]*moira.MetricValue{
				metric1: {},
				metric2: {
					&moira.MetricValue{
						RetentionTimestamp: tsOlder,
						Timestamp:          tsOlder,
					},
				},
			})

			err = dataBase.CleanUpAbandonedRetentions()
			So(err, ShouldBeNil)

			Convey("metric1 retention key shouldn't be and metric2 retention key should be in database", func() {
				isMetric1RetentionExists := client.Exists(dataBase.context, metricRetentionKey(metric1)).Val() == 1
				So(isMetric1RetentionExists, ShouldBeFalse)

				isMetric2RetentionExists := client.Exists(dataBase.context, metricRetentionKey(metric2)).Val() == 1
				So(isMetric2RetentionExists, ShouldBeTrue)
			})
		})
	})
}

func TestRemoveMetricsByPrefix(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test", true)
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()
	client := *dataBase.client
	const pattern = "my.test.*.metric*"

	Convey("Given metrics with pattern my.test.*", t, func() {
		for i := 1; i <= 10; i++ {
			err := dataBase.SaveMetrics(
				map[string]*moira.MatchedMetric{
					fmt.Sprintf("my.test.super.metric%d", i): {
						Patterns:           []string{pattern},
						Metric:             fmt.Sprintf("my.test.super.metric%d", i),
						Retention:          10,
						RetentionTimestamp: 10,
						Timestamp:          5,
						Value:              1,
					},
				})
			So(err, ShouldBeNil)
		}

		for i := 1; i <= 10; i++ {
			err := dataBase.SaveMetrics(
				map[string]*moira.MatchedMetric{
					fmt.Sprintf("my.test.mega.metric%d", i): {
						Patterns:           []string{pattern},
						Metric:             fmt.Sprintf("my.test.mega.metric%d", i),
						Retention:          10,
						RetentionTimestamp: 10,
						Timestamp:          5,
						Value:              1,
					},
				})
			So(err, ShouldBeNil)
		}

		result := client.Keys(dataBase.context, "moira-metric-data:my.test*").Val()
		So(len(result), ShouldResemble, 20)
		result = client.Keys(dataBase.context, "moira-metric-retention:my.test*").Val()
		So(len(result), ShouldResemble, 20)

		patternMetricsCount := client.SCard(dataBase.context, "moira-pattern-metrics:my.test.*.metric*").Val()
		So(patternMetricsCount, ShouldResemble, int64(20))

		Convey("When remove metrics by prefix my.test.super. was called", func() {
			err := dataBase.RemoveMetricsByPrefix("my.test.super.")
			So(err, ShouldBeNil)

			Convey("No metric data for metrics with this prefix should not exist", func() {
				result = client.Keys(dataBase.context, "moira-metric-data:my.test*").Val()
				So(len(result), ShouldResemble, 10)
				result = client.Keys(dataBase.context, "moira-metric-retention:my.test*").Val()
				So(len(result), ShouldResemble, 10)
				result = client.Keys(dataBase.context, "moira-metric-data:my.test.mega.*").Val()
				So(len(result), ShouldResemble, 10)
				result = client.Keys(dataBase.context, "moira-metric-retention:my.test.mega.*").Val()
				So(len(result), ShouldResemble, 10)
				patternMetricsCount := client.SCard(dataBase.context, "moira-pattern-metrics:my.test.*.metric*").Val()
				So(patternMetricsCount, ShouldResemble, int64(10))
			})
		})
	})
}

func TestRemoveAllMetrics(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test", true)
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()
	client := *dataBase.client
	const pattern = "my.test.*.metric*"

	Convey("Given metrics with pattern my.test.*", t, func() {
		for i := 1; i <= 10; i++ {
			err := dataBase.SaveMetrics(
				map[string]*moira.MatchedMetric{
					fmt.Sprintf("my.test.super.metric%d", i): {
						Patterns:           []string{pattern},
						Metric:             fmt.Sprintf("my.test.super.metric%d", i),
						Retention:          10,
						RetentionTimestamp: 10,
						Timestamp:          5,
						Value:              1,
					},
				})
			So(err, ShouldBeNil)
		}

		for i := 1; i <= 10; i++ {
			err := dataBase.SaveMetrics(
				map[string]*moira.MatchedMetric{
					fmt.Sprintf("my.test.mega.metric%d", i): {
						Patterns:           []string{pattern},
						Metric:             fmt.Sprintf("my.test.mega.metric%d", i),
						Retention:          10,
						RetentionTimestamp: 10,
						Timestamp:          5,
						Value:              1,
					},
				})
			So(err, ShouldBeNil)
		}

		result := client.Keys(dataBase.context, "moira-metric-data:my.test*").Val()
		So(len(result), ShouldResemble, 20)
		result = client.Keys(dataBase.context, "moira-metric-retention:my.test*").Val()
		So(len(result), ShouldResemble, 20)

		patternMetricsCount := client.SCard(dataBase.context, "moira-pattern-metrics:my.test.*.metric*").Val()
		So(patternMetricsCount, ShouldResemble, int64(20))

		Convey("When remove all metrics was called", func() {
			err := dataBase.RemoveAllMetrics()
			So(err, ShouldBeNil)

			Convey("No metric data should not exist", func() {
				result = client.Keys(dataBase.context, "moira-metric-data:*").Val()
				So(len(result), ShouldResemble, 0)
				result = client.Keys(dataBase.context, "moira-metric-retention:*").Val()
				So(len(result), ShouldResemble, 0)
				result = client.Keys(dataBase.context, "moira-pattern-metrics*").Val()
				So(len(result), ShouldResemble, 0)
			})
		})
	})
}
