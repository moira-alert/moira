package conversion

import (
	"testing"

	metricSource "github.com/moira-alert/moira/metric_source"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_isOneMetricMap(t *testing.T) {
	type args struct {
		metrics map[string]metricSource.MetricData
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 string
	}{
		{
			name: "is one metric map",
			args: args{
				metrics: map[string]metricSource.MetricData{
					"metric.test.1": {},
				},
			},
			want:  true,
			want1: "metric.test.1",
		},
		{
			name: "is not one metric map",
			args: args{
				metrics: map[string]metricSource.MetricData{
					"metric.test.1": {},
					"metric.test.2": {},
				},
			},
			want:  false,
			want1: "",
		},
	}
	Convey("metrics map", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				ok, metricName := isOneMetricMap(tt.args.metrics)
				So(ok, ShouldResemble, tt.want)
				So(metricName, ShouldResemble, tt.want1)
			})
		}
	})
}

func TestMetricName(t *testing.T) {
	type args struct {
		metrics map[string]metricSource.MetricData
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "origin is empty",
			args: args{
				metrics: map[string]metricSource.MetricData{},
			},
			want: "",
		},
		{
			name: "origin is not empty and all metrics have same name",
			args: args{
				metrics: map[string]metricSource.MetricData{
					"t1": metricSource.MetricData{Name: "metric.test.1"},
					"t2": metricSource.MetricData{Name: "metric.test.1"},
				},
			},
			want: "metric.test.1",
		},
		{
			name: "origin is not empty and metrics have different names",
			args: args{
				metrics: map[string]metricSource.MetricData{
					"t1": metricSource.MetricData{Name: "metric.test.2"},
					"t2": metricSource.MetricData{Name: "metric.test.1"},
				},
			},
			want: "metric.test.1",
		},
	}
	Convey("MetricName", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				actual := MetricName(tt.args.metrics)
				So(actual, ShouldResemble, tt.want)
			})
		}
	})
}

func TestGetRelations(t *testing.T) {
	type args struct {
		metrics map[string]metricSource.MetricData
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "origin is empty",
			args: args{

				metrics: map[string]metricSource.MetricData{},
			},
			want: map[string]string{},
		},
		{
			name: "origin is not empty",
			args: args{
				metrics: map[string]metricSource.MetricData{
					"t1": metricSource.MetricData{Name: "metric.test.1"},
					"t2": metricSource.MetricData{Name: "metric.test.2"},
				},
			},
			want: map[string]string{
				"t1": "metric.test.1",
				"t2": "metric.test.2",
			},
		},
	}
	Convey("GetRelations", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				actual := GetRelations(tt.args.metrics)
				So(actual, ShouldResemble, tt.want)
			})
		}
	})
}

func TestMerge(t *testing.T) {
	type args struct {
		metrics map[string]metricSource.MetricData
		other   map[string]metricSource.MetricData
	}
	tests := []struct {
		name string
		args args
		want map[string]metricSource.MetricData
	}{
		{
			name: "origin and other are empty",
			args: args{
				metrics: map[string]metricSource.MetricData{},
				other:   map[string]metricSource.MetricData{},
			},
			want: map[string]metricSource.MetricData{},
		},
		{
			name: "origin is empty and other is not",
			args: args{
				metrics: map[string]metricSource.MetricData{},
				other:   map[string]metricSource.MetricData{"t1": metricSource.MetricData{Name: "metric.test.1"}},
			},
			want: map[string]metricSource.MetricData{"t1": metricSource.MetricData{Name: "metric.test.1"}},
		},
		{
			name: "origin is not empty and other is empty",
			args: args{
				metrics: map[string]metricSource.MetricData{"t1": metricSource.MetricData{Name: "metric.test.1"}},
				other:   map[string]metricSource.MetricData{},
			},
			want: map[string]metricSource.MetricData{"t1": metricSource.MetricData{Name: "metric.test.1"}},
		},
		{
			name: "origin and other have same targets",
			args: args{
				metrics: map[string]metricSource.MetricData{"t1": metricSource.MetricData{Name: "metric.test.1"}},
				other:   map[string]metricSource.MetricData{"t1": metricSource.MetricData{Name: "metric.test.2"}},
			},
			want: map[string]metricSource.MetricData{"t1": metricSource.MetricData{Name: "metric.test.2"}},
		},
	}

	Convey("Merge", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				actual := Merge(tt.args.metrics, tt.args.other)
				So(actual, ShouldResemble, tt.want)
			})
		}
	})
}

func TestHasOnlyWildcards(t *testing.T) {
	type args struct {
		metrics map[string][]metricSource.MetricData
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "does not have wildcards",
			args: args{
				metrics: map[string][]metricSource.MetricData{
					"t1": {
						{Name: "metric.test.1", Wildcard: false},
						{Name: "metric.test.2", Wildcard: false},
					},
				},
			},
			want: false,
		},
		{
			name: "one target has wildcards",
			args: args{
				metrics: map[string][]metricSource.MetricData{
					"t1": {
						{Name: "metric.test.1", Wildcard: true},
						{Name: "metric.test.2", Wildcard: true},
					},
					"t2": {
						{Name: "metric.test.1", Wildcard: false},
						{Name: "metric.test.2", Wildcard: true},
					},
				},
			},
			want: false,
		},
		{
			name: "has only wildcards",
			args: args{
				metrics: map[string][]metricSource.MetricData{
					"t1": {
						{Name: "metric.test.1", Wildcard: true},
						{Name: "metric.test.2", Wildcard: true},
					},
					"t2": {
						{Name: "metric.test.1", Wildcard: true},
						{Name: "metric.test.2", Wildcard: true},
					},
				},
			},
			want: true,
		},
	}
	Convey("HasOnlyWildcards", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				actual := HasOnlyWildcards(tt.args.metrics)
				So(actual, ShouldResemble, tt.want)
			})
		}
	})
}
