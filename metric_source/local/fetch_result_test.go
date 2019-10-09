package local

import (
	"testing"

	metricSource "github.com/moira-alert/moira/metric_source"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCreateEmptyFetchResult(t *testing.T) {
	Convey("Just create fetch empty fetch result", t, func() {
		So(*(CreateEmptyFetchResult()), ShouldResemble, FetchResult{
			MetricsData: make([]metricSource.MetricData, 0),
			Patterns:    make([]string, 0),
			Metrics:     make([]string, 0),
		})
	})
}

func TestFetchResult_GetMetricsData(t *testing.T) {
	Convey("Get empty metric data", t, func() {
		fetchResult := &FetchResult{
			MetricsData: make([]metricSource.MetricData, 0),
			Patterns:    make([]string, 0),
			Metrics:     make([]string, 0),
		}
		So(fetchResult.GetMetricsData(), ShouldBeEmpty)
	})

	Convey("Get not empty metric data", t, func() {
		fetchResult := &FetchResult{
			MetricsData: []metricSource.MetricData{*metricSource.MakeMetricData("123", []float64{1, 2, 3}, 60, 0)},
			Patterns:    make([]string, 0),
			Metrics:     make([]string, 0),
		}
		So(fetchResult.GetMetricsData(), ShouldHaveLength, 1)
	})
}

func TestFetchResult_GetPatternMetrics(t *testing.T) {
	Convey("Get empty pattern metrics", t, func() {
		fetchResult := &FetchResult{
			MetricsData: make([]metricSource.MetricData, 0),
			Patterns:    make([]string, 0),
			Metrics:     make([]string, 0),
		}
		actual, err := fetchResult.GetPatternMetrics()
		So(actual, ShouldBeEmpty)
		So(err, ShouldBeNil)
	})

	Convey("Get not empty metric data", t, func() {
		fetchResult := &FetchResult{
			MetricsData: []metricSource.MetricData{*metricSource.MakeMetricData("123", []float64{1, 2, 3}, 60, 0)},
			Patterns:    make([]string, 0),
			Metrics:     []string{"123"},
		}
		actual, err := fetchResult.GetPatternMetrics()
		So(actual, ShouldHaveLength, 1)
		So(err, ShouldBeNil)
	})
}

func TestFetchResult_GetPatterns(t *testing.T) {
	Convey("Get empty pattern metrics", t, func() {
		fetchResult := &FetchResult{
			MetricsData: make([]metricSource.MetricData, 0),
			Patterns:    make([]string, 0),
			Metrics:     make([]string, 0),
		}
		actual, err := fetchResult.GetPatterns()
		So(actual, ShouldBeEmpty)
		So(err, ShouldBeNil)
	})

	Convey("Get not empty metric data", t, func() {
		fetchResult := &FetchResult{
			MetricsData: []metricSource.MetricData{*metricSource.MakeMetricData("123", []float64{1, 2, 3}, 60, 0)},
			Patterns:    []string{"123"},
			Metrics:     []string{"123"},
		}
		actual, err := fetchResult.GetPatterns()
		So(actual, ShouldHaveLength, 1)
		So(err, ShouldBeNil)
	})
}
