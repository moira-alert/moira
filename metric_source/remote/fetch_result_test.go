package remote

import (
	"testing"

	metricSource "github.com/moira-alert/moira/metric_source"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFetchResult(t *testing.T) {
	Convey("Get empty metric data", t, func() {
		fetchResult := FetchResult{
			MetricsData: make([]metricSource.MetricData, 0),
		}
		So(fetchResult.GetMetricsData(), ShouldBeEmpty)
		patterns, err := fetchResult.GetPatterns()
		So(patterns, ShouldBeEmpty)
		So(err, ShouldNotBeEmpty)
		metrics, err := fetchResult.GetPatternMetrics()
		So(metrics, ShouldBeEmpty)
		So(err, ShouldNotBeEmpty)
	})

	Convey("Get not empty metric data", t, func() {
		fetchResult := &FetchResult{
			MetricsData: []metricSource.MetricData{*metricSource.MakeMetricData("123", []float64{1, 2, 3}, 60, 0)},
		}
		So(fetchResult.GetMetricsData(), ShouldHaveLength, 1)
		patterns, err := fetchResult.GetPatterns()
		So(patterns, ShouldBeEmpty)
		So(err, ShouldNotBeEmpty)
		metrics, err := fetchResult.GetPatternMetrics()
		So(metrics, ShouldBeEmpty)
		So(err, ShouldNotBeEmpty)
	})
}
