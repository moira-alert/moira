package local

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/moira-alert/moira"

	"github.com/moira-alert/moira/clock"
	"github.com/moira-alert/moira/database/redis"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"

	. "github.com/smartystreets/goconvey/convey"
)

type metricMock struct {
	values   []float64
	patterns []string
}

type testCase struct {
	metrics           map[string]metricMock
	from              int64
	retention         int64
	target            string
	expected          map[string][]float64
	expectedWildcards map[string]bool
}

func saveMetrics(database moira.Database, metrics map[string]metricMock, now, retention int64) error {
	maxValues := 0
	for _, m := range metrics {
		if len(m.values) > maxValues {
			maxValues = len(m.values)
		}
	}

	timeStart := now - retention*int64(maxValues-1)
	for i := range maxValues {
		time := timeStart + int64(i)*retention

		metricsBuffer := make([]*moira.MatchedMetric, 0, len(metrics))

		for name, metric := range metrics {
			if len(metric.values) <= i {
				continue
			}

			metricsBuffer = append(metricsBuffer, &moira.MatchedMetric{
				Metric:             name,
				Patterns:           metric.patterns,
				Value:              metric.values[i],
				Timestamp:          time,
				RetentionTimestamp: time,
				Retention:          int(retention),
			})
		}

		err := database.SaveMetrics(metricsBuffer)
		if err != nil {
			return err
		}
	}

	return nil
}

func TestLocalSourceWithDatabaseWildcards(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test", true) // nolint: govet
	database := redis.NewTestDatabase(logger, clock.NewSystemClock())
	localSource := Create(database)

	defer database.Flush()

	retention := int64(60)
	now := floorToMultiplier(time.Now().Unix(), retention) - 2

	testCases := []testCase{
		{
			metrics: map[string]metricMock{
				"metric1": {
					values:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
					patterns: []string{"pattern"},
				},
				"metric2": {
					values:   []float64{5.0, 4.0, 3.0, 2.0, 1.0},
					patterns: []string{"pattern"},
				},
			},
			from:      now - retention*4,
			retention: retention,
			target:    "pattern",
			expectedWildcards: map[string]bool{
				"metric1": false,
				"metric2": false,
			},
		},
		{
			metrics: map[string]metricMock{
				"metric1": {
					values:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
					patterns: []string{"pattern1"},
				},
				"metric2": {
					values:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
					patterns: []string{"pattern1"},
				},
			},
			from:      now - retention*4,
			retention: retention,
			target:    "divideSeries(pattern1, pattern2)",
			expectedWildcards: map[string]bool{
				"divideSeries(metric1,pattern2)": false,
				"divideSeries(metric2,pattern2)": false,
			},
		},
		{
			metrics:   map[string]metricMock{},
			from:      now - retention*4,
			retention: retention,
			target:    "pattern",
			expectedWildcards: map[string]bool{
				"pattern": true,
			},
		},
	}

	Convey("Run test cases", t, func() {
		for idx, testCase := range testCases {
			Convey(fmt.Sprintf("suite %d, Target '%s'", idx, testCase.target), func() {
				database.Flush()

				err := saveMetrics(database, testCase.metrics, now, testCase.retention)
				So(err, ShouldBeNil)

				result, err := localSource.Fetch(testCase.target, testCase.from, now, true)
				So(err, ShouldBeNil)

				resultData := result.GetMetricsData()

				wildcardResultMap := map[string]bool{}
				for _, data := range resultData {
					wildcardResultMap[data.Name] = data.Wildcard
				}

				So(wildcardResultMap, shouldEqualIfNaNsEqual, testCase.expectedWildcards)
			})
		}
	})
}

func TestLocalSourceWithDatabase(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test", true) // nolint: govet
	database := redis.NewTestDatabase(logger, clock.NewSystemClock())
	localSource := Create(database)

	defer database.Flush()

	retention := int64(60)
	now := floorToMultiplier(time.Now().Unix(), retention)
	nan := math.NaN()

	testCases := []testCase{
		{
			metrics: map[string]metricMock{
				"metric1": {
					values:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
					patterns: []string{"pattern"},
				},
				"metric2": {
					values:   []float64{5.0, 4.0, 3.0, 2.0, 1.0},
					patterns: []string{"pattern"},
				},
			},
			from:      now - retention*4,
			retention: retention,
			target:    "pattern",
			expected: map[string][]float64{
				"metric1": {1.0, 2.0, 3.0, 4.0, 5.0},
				"metric2": {5.0, 4.0, 3.0, 2.0, 1.0},
			},
		},
		{
			metrics: map[string]metricMock{
				"metric1": {
					values:   []float64{1.0, 3.0, 1.0, 3.0, 1.0, 3.0},
					patterns: []string{"pattern"},
				},
			},
			from:      now - retention*4,
			retention: retention,
			target:    "movingAverage(pattern, 2)",
			expected: map[string][]float64{
				"movingAverage(metric1,2)": {2.0, 2.0, 2.0, 2.0, 2.0},
			},
		},
		{
			metrics: map[string]metricMock{
				"metric1": {
					values:   []float64{1.0, nan, 2.0, nan, nan},
					patterns: []string{"pattern"},
				},
			},
			from:      now - retention*4,
			retention: retention,
			target:    "keepLastValue(pattern, 1)",
			expected: map[string][]float64{
				"keepLastValue(metric1,1)": {1.0, 1.0, 2.0, 2.0, nan},
			},
		},
		{
			metrics: map[string]metricMock{
				"metric.1.foo": {
					values:   []float64{1.0, 2.0, 1.0, 2.0, 1.0},
					patterns: []string{"metric.*.*", "metric.1.foo"},
				},
				"metric.1.bar": {
					values:   []float64{-1.0, -2.0, -3.0, -4.0, -5.0},
					patterns: []string{"metric.*.*", "metric.1.bar"},
				},
				"metric.2.foo": {
					values:   []float64{3.0, 2.0, 3.0, 2.0, 3.0},
					patterns: []string{"metric.*.*", "metric.2.foo"},
				},
				"metric.2.bar": {
					values:   []float64{-1.0, -2.0, -3.0, -4.0, -5.0},
					patterns: []string{"metric.*.*", "metric.2.bar"},
				},
			},
			from:      now - retention*4,
			retention: retention,
			target:    `applyByNode(metric.*.*, 1, "movingMax(%.foo, '2m')")`,
			expected: map[string][]float64{
				"movingMax(metric.1.foo,'2m')": {1.0, 2.0, 2.0, 2.0, 2.0},
				"movingMax(metric.2.foo,'2m')": {3.0, 3.0, 3.0, 3.0, 3.0},
			},
		},
		{
			metrics: map[string]metricMock{
				"metric.foo": {
					values:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
					patterns: []string{"metric.*"},
				},
			},
			from:      now - retention*4,
			retention: retention,
			target:    "aliasByNode(metric.*, 1)",
			expected: map[string][]float64{
				"foo": {1.0, 2.0, 3.0, 4.0, 5.0},
			},
		},
		{
			metrics: map[string]metricMock{
				"metric.foo": {
					values:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
					patterns: []string{"metric.*"},
				},
			},
			from:      now - retention*4,
			retention: retention,
			target:    "aliasByNode(metric.*, 2)",
			expected: map[string][]float64{
				"": {1.0, 2.0, 3.0, 4.0, 5.0},
			},
		},
		{
			metrics: map[string]metricMock{
				"metric.foo": {
					values:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
					patterns: []string{"metric.*"},
				},
			},
			from:      now - retention*4,
			retention: retention,
			target:    "consolidateBy(metric.*, 'max')",
			expected: map[string][]float64{
				`consolidateBy(metric.foo,"max")`: {1.0, 2.0, 3.0, 4.0, 5.0},
			},
		},
		{
			metrics: map[string]metricMock{
				"metric.1": {
					values:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
					patterns: []string{"metric.*"},
				},
				"metric.2": {
					values:   []float64{-1.0, 2.0, 3.0, 4.0, 5.0},
					patterns: []string{"metric.*"},
				},
				"metric.3": {
					values:   []float64{-1.0, -2.0, -3.0, -4.0, -5.0},
					patterns: []string{"metric.*"},
				},
			},
			from:      now - retention*4,
			retention: retention,
			target:    "minimumBelow(metric.*, 0)",
			expected: map[string][]float64{
				"metric.2": {-1.0, 2.0, 3.0, 4.0, 5.0},
				"metric.3": {-1.0, -2.0, -3.0, -4.0, -5.0},
			},
		},
		{
			metrics: map[string]metricMock{
				"metric.foo.1": {
					values:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
					patterns: []string{"metric.*.*"},
				},
				"metric.foo.2": {
					values:   []float64{1.5, 2.5, 3.5, 4.5, 5.5},
					patterns: []string{"metric.*.*"},
				},
			},
			from:      now - retention*4,
			retention: retention,
			target:    "groupByNode(metric.*.*, 1, 'sumSeries')",
			expected: map[string][]float64{
				"foo": {2.5, 4.5, 6.5, 8.5, 10.5},
			},
		},
		{
			metrics: map[string]metricMock{
				"metric.foo.1": {
					values:   []float64{1.5, 2.5, 3.5, 4.5, 5.5},
					patterns: []string{"metric.*.*"},
				},
				"metric.foo.2": {
					values:   []float64{1.5, 2.5, 3.5, 4.5, 5.5},
					patterns: []string{"metric.*.*"},
				},
			},
			from:      now - retention*4,
			retention: retention,
			target:    "groupByNode(metric.*.*, 1, 'unique')",
			expected: map[string][]float64{
				"foo": {1.5, 2.5, 3.5, 4.5, 5.5},
			},
		},
		{
			metrics: map[string]metricMock{
				"metric.foo": {
					values:   []float64{0, 1, 2, 3, 4, 5},
					patterns: []string{"metric.*"},
				},
			},
			from:      now - retention*5,
			retention: retention,
			target:    "hitcount(metric.*, '2m')",
			expected: map[string][]float64{
				"hitcount(metric.foo,'2m')": {60, 300, 540},
			},
		},
		{
			metrics: map[string]metricMock{
				"metric.foo": {
					values:   []float64{1, nan, 3, nan, nan, 6},
					patterns: []string{"metric.*"},
				},
			},
			from:      now - retention*5,
			retention: retention,
			target:    "interpolate(metric.*, 1)",
			expected: map[string][]float64{
				"interpolate(metric.foo)": {1, 2, 3, nan, nan, 6},
			},
		},
		{
			metrics: map[string]metricMock{
				"metric.foo": {
					values:   []float64{1, nan, 3, nan, nan, 6},
					patterns: []string{"metric.*"},
				},
			},
			from:      now - retention*5,
			retention: retention,
			target:    "interpolate(metric.*)",
			expected: map[string][]float64{
				"interpolate(metric.foo)": {1, 2, 3, 4, 5, 6},
			},
		},
		{
			metrics: map[string]metricMock{
				"metric.foo": {
					values:   []float64{1, 2, 3, 4, 5, 6},
					patterns: []string{"metric.*"},
				},
			},
			from:      now - retention*5,
			retention: retention,
			target:    "smartSummarize(metric.*, '2m', 'average')",
			expected: map[string][]float64{
				"smartSummarize(metric.foo,'2m','average')": {1.5, 3.5, 5.5},
			},
		},
		{
			metrics: map[string]metricMock{
				"metric.foo": {
					values:   []float64{1.5, 2, 3, 4, 5, 6.5},
					patterns: []string{"metric.*"},
				},
			},
			from:      now - retention*5,
			retention: retention,
			target:    "smartSummarize(metric.*, '3m', 'median')",
			expected: map[string][]float64{
				"smartSummarize(metric.foo,'3m','median')": {2, 5},
			},
		},
	}

	Convey("Run test cases", t, func() {
		for _, testCase := range testCases {
			Convey(fmt.Sprintf("Target '%s'", testCase.target), func() {
				database.Flush()

				err := saveMetrics(database, testCase.metrics, now, testCase.retention)
				So(err, ShouldBeNil)

				result, err := localSource.Fetch(testCase.target, testCase.from, now, true)
				So(err, ShouldBeNil)

				resultData := result.GetMetricsData()
				resultMap := map[string][]float64{}

				for _, data := range resultData {
					resultMap[data.Name] = data.Values
				}

				So(resultMap, shouldEqualIfNaNsEqual, testCase.expected)
			})
		}
	})
}
