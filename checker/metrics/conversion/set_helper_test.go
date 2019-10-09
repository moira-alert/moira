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
		want setHelper
	}{
		{
			name: "is empty",
			args: args{
				metrics: TriggerTargetMetrics{},
			},
			want: setHelper{},
		},
		{
			name: "is not empty",
			args: args{
				metrics: TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.name.1"},
				},
			},
			want: setHelper{"metric.test.1": true},
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
		other setHelper
	}
	tests := []struct {
		name string
		h    setHelper
		args args
		want setHelper
	}{
		{
			name: "Both empty",
			h:    setHelper{},
			args: args{
				other: setHelper{},
			},
			want: setHelper{},
		},
		{
			name: "Target is empty, other is not empty",
			h:    setHelper{},
			args: args{
				other: setHelper{"metric.test.1": true},
			},
			want: setHelper{"metric.test.1": true},
		},
		{
			name: "Target is not empty, other is empty",
			h:    setHelper{"metric.test.1": true},
			args: args{
				other: setHelper{},
			},
			want: setHelper{"metric.test.1": true},
		},
		{
			name: "Both are not empty",
			h:    setHelper{"metric.test.1": true},
			args: args{
				other: setHelper{"metric.test.2": true},
			},
			want: setHelper{"metric.test.1": true, "metric.test.2": true},
		},
		{
			name: "Both are not empty and have same names",
			h:    setHelper{"metric.test.1": true, "metric.test.2": true},
			args: args{
				other: setHelper{"metric.test.2": true, "metric.test.3": true},
			},
			want: setHelper{"metric.test.1": true, "metric.test.2": true, "metric.test.3": true},
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
		other setHelper
	}
	tests := []struct {
		name string
		h    setHelper
		args args
		want setHelper
	}{
		{
			name: "both have same elements",
			h:    setHelper{"t1": true, "t2": true},
			args: args{
				other: setHelper{"t1": true, "t2": true},
			},
			want: setHelper{},
		},
		{
			name: "other have additional values",
			h:    setHelper{"t1": true, "t2": true},
			args: args{
				other: setHelper{"t1": true, "t2": true, "t3": true},
			},
			want: setHelper{"t3": true},
		},
		{
			name: "origin have additional values",
			h:    setHelper{"t1": true, "t2": true, "t3": true},
			args: args{
				other: setHelper{"t1": true, "t2": true},
			},
			want: setHelper{},
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
