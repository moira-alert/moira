package conversion

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewFetchedTargetMetricsWithCapacity(t *testing.T) {
	Convey("NewFetchedTargetMetricsWithCapacity", t, func() {
		Convey("call", func() {
			capacity := 10
			actual := NewFetchedTargetMetricsWithCapacity(capacity)
			So(actual, ShouldNotBeNil)
			So(actual, ShouldHaveLength, 0)
			So(cap(actual), ShouldEqual, capacity)
		})
	})
}

func TestFetchedTargetMetrics_CleanWildcards(t *testing.T) {
	tests := []struct {
		name string
		m    FetchedTargetMetrics
		want FetchedTargetMetrics
	}{
		{
			name: "does not have wildcards",
			m: FetchedTargetMetrics{
				{Name: "metric.test.1", Wildcard: false},
			},
			want: FetchedTargetMetrics{
				{Name: "metric.test.1", Wildcard: false},
			},
		},
		{
			name: "has wildcards",
			m: FetchedTargetMetrics{
				{Name: "metric.test.1", Wildcard: false},
				{Name: "metric.test.2", Wildcard: true},
			},
			want: FetchedTargetMetrics{
				{Name: "metric.test.1", Wildcard: false},
			},
		},
	}
	Convey("FetchedTargetMetrics", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				actual := tt.m.CleanWildcards()
				So(actual, ShouldResemble, tt.want)
			})
		}
	})
}

func TestFetchedTargetMetrics_Deduplicate(t *testing.T) {
	tests := []struct {
		name             string
		m                FetchedTargetMetrics
		wantDeduplicated FetchedTargetMetrics
		wantDuplicates   []string
	}{
		{
			name: "does not have duplicates",
			m: FetchedTargetMetrics{
				{Name: "metric.test.1"},
				{Name: "metric.test.2"},
			},
			wantDeduplicated: FetchedTargetMetrics{
				{Name: "metric.test.1"},
				{Name: "metric.test.2"},
			},
			wantDuplicates: nil,
		},
		{
			name: "has duplicates",
			m: FetchedTargetMetrics{
				{Name: "metric.test.1"},
				{Name: "metric.test.1"},
				{Name: "metric.test.2"},
			},
			wantDeduplicated: FetchedTargetMetrics{
				{Name: "metric.test.1"},
				{Name: "metric.test.2"},
			},
			wantDuplicates: []string{"metric.test.1"},
		},
		{
			name: "has multiple duplicates",
			m: FetchedTargetMetrics{
				{Name: "metric.test.1"},
				{Name: "metric.test.1"},
				{Name: "metric.test.1"},
				{Name: "metric.test.2"},
			},
			wantDeduplicated: FetchedTargetMetrics{
				{Name: "metric.test.1"},
				{Name: "metric.test.2"},
			},
			wantDuplicates: []string{"metric.test.1", "metric.test.1"},
		},
	}
	Convey("FetchedTargetMetrics", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				deduplicated, duplicates := tt.m.Deduplicate()
				So(deduplicated, ShouldResemble, tt.wantDeduplicated)
				So(duplicates, ShouldResemble, tt.wantDuplicates)
			})
		}
	})
}

func TestNewFetchedMetricsWithCapacity(t *testing.T) {
	Convey("NewNewFetchedMetricsWithCapacity", t, func() {
		Convey("call", func() {
			capacity := 10
			actual := NewFetchedMetricsWithCapacity(capacity)
			So(actual, ShouldNotBeNil)
			So(actual, ShouldHaveLength, 0)
		})
	})
}
