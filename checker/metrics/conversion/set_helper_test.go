package conversion

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_newSetHelperFromTriggerTargetMetrics(t *testing.T) {
	type args struct {
		metrics TriggerTargetMetrics
	}
	tests := []struct {
		name string
		args args
		want set[string]
	}{
		{
			name: "is empty",
			args: args{
				metrics: TriggerTargetMetrics{},
			},
			want: set[string]{},
		},
		{
			name: "is not empty",
			args: args{
				metrics: TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.name.1"},
				},
			},
			want: set[string]{"metric.test.1": void},
		},
	}

	Convey("TriggerPatterMetrics", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				actual := newSetHelperFromTriggerTargetMetrics(tt.args.metrics)
				So(actual, ShouldResemble, tt.want)
			})
		}
	})
}

func Test_setHelper_union(t *testing.T) {
	type args struct {
		other set[string]
	}
	tests := []struct {
		name string
		h    set[string]
		args args
		want set[string]
	}{
		{
			name: "Both empty",
			h:    set[string]{},
			args: args{
				other: set[string]{},
			},
			want: set[string]{},
		},
		{
			name: "Target is empty, other is not empty",
			h:    set[string]{},
			args: args{
				other: set[string]{"metric.test.1": void},
			},
			want: set[string]{"metric.test.1": void},
		},
		{
			name: "Target is not empty, other is empty",
			h:    set[string]{"metric.test.1": void},
			args: args{
				other: set[string]{},
			},
			want: set[string]{"metric.test.1": void},
		},
		{
			name: "Both are not empty",
			h:    set[string]{"metric.test.1": void},
			args: args{
				other: set[string]{"metric.test.2": void},
			},
			want: set[string]{"metric.test.1": void, "metric.test.2": void},
		},
		{
			name: "Both are not empty and have same names",
			h:    set[string]{"metric.test.1": void, "metric.test.2": void},
			args: args{
				other: set[string]{"metric.test.2": void, "metric.test.3": void},
			},
			want: set[string]{"metric.test.1": void, "metric.test.2": void, "metric.test.3": void},
		},
	}
	Convey("union", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				actual := tt.h.union(tt.args.other)
				So(actual, ShouldResemble, tt.want)
			})
		}
	})
}

func Test_setHelper_diff(t *testing.T) {
	type args struct {
		other set[string]
	}
	tests := []struct {
		name string
		h    set[string]
		args args
		want set[string]
	}{
		{
			name: "both have same elements",
			h:    set[string]{"t1": void, "t2": void},
			args: args{
				other: set[string]{"t1": void, "t2": void},
			},
			want: set[string]{},
		},
		{
			name: "other have additional values",
			h:    set[string]{"t1": void, "t2": void},
			args: args{
				other: set[string]{"t1": void, "t2": void, "t3": void},
			},
			want: set[string]{"t3": void},
		},
		{
			name: "origin have additional values",
			h:    set[string]{"t1": void, "t2": void, "t3": void},
			args: args{
				other: set[string]{"t1": void, "t2": void},
			},
			want: set[string]{},
		},
	}
	Convey("diff", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				actual := tt.h.diff(tt.args.other)
				So(actual, ShouldResemble, tt.want)
			})
		}
	})
}
