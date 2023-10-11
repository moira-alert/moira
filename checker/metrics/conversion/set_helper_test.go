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
		want setᐸstringᐳ
	}{
		{
			name: "is empty",
			args: args{
				metrics: TriggerTargetMetrics{},
			},
			want: setᐸstringᐳ{},
		},
		{
			name: "is not empty",
			args: args{
				metrics: TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.name.1"},
				},
			},
			want: setᐸstringᐳ{"metric.test.1": void},
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
		other setᐸstringᐳ
	}
	tests := []struct {
		name string
		h    setᐸstringᐳ
		args args
		want setᐸstringᐳ
	}{
		{
			name: "Both empty",
			h:    setᐸstringᐳ{},
			args: args{
				other: setᐸstringᐳ{},
			},
			want: setᐸstringᐳ{},
		},
		{
			name: "Target is empty, other is not empty",
			h:    setᐸstringᐳ{},
			args: args{
				other: setᐸstringᐳ{"metric.test.1": void},
			},
			want: setᐸstringᐳ{"metric.test.1": void},
		},
		{
			name: "Target is not empty, other is empty",
			h:    setᐸstringᐳ{"metric.test.1": void},
			args: args{
				other: setᐸstringᐳ{},
			},
			want: setᐸstringᐳ{"metric.test.1": void},
		},
		{
			name: "Both are not empty",
			h:    setᐸstringᐳ{"metric.test.1": void},
			args: args{
				other: setᐸstringᐳ{"metric.test.2": void},
			},
			want: setᐸstringᐳ{"metric.test.1": void, "metric.test.2": void},
		},
		{
			name: "Both are not empty and have same names",
			h:    setᐸstringᐳ{"metric.test.1": void, "metric.test.2": void},
			args: args{
				other: setᐸstringᐳ{"metric.test.2": void, "metric.test.3": void},
			},
			want: setᐸstringᐳ{"metric.test.1": void, "metric.test.2": void, "metric.test.3": void},
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
		other setᐸstringᐳ
	}
	tests := []struct {
		name string
		h    setᐸstringᐳ
		args args
		want setᐸstringᐳ
	}{
		{
			name: "both have same elements",
			h:    setᐸstringᐳ{"t1": void, "t2": void},
			args: args{
				other: setᐸstringᐳ{"t1": void, "t2": void},
			},
			want: setᐸstringᐳ{},
		},
		{
			name: "other have additional values",
			h:    setᐸstringᐳ{"t1": void, "t2": void},
			args: args{
				other: setᐸstringᐳ{"t1": void, "t2": void, "t3": void},
			},
			want: setᐸstringᐳ{"t3": void},
		},
		{
			name: "origin have additional values",
			h:    setᐸstringᐳ{"t1": void, "t2": void, "t3": void},
			args: args{
				other: setᐸstringᐳ{"t1": void, "t2": void},
			},
			want: setᐸstringᐳ{},
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
