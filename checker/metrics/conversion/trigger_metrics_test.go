package conversion

import (
	"math"
	"testing"

	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_newTriggerTargetMetricsWithCapacity(t *testing.T) {
	Convey("newTriggerTargetMetricsWithCapacity", t, func() {
		Convey("call", func() {
			capacity := 10
			actual := newTriggerTargetMetricsWithCapacity(capacity)
			So(actual, ShouldNotBeNil)
			So(actual, ShouldHaveLength, 0)
		})
	})
}

func TestNewTriggerTargetMetrics(t *testing.T) {
	Convey("NewTriggerTargetMetrics", t, func() {
		fetched := FetchedTargetMetrics{
			{Name: "metric.test.1"},
			{Name: "metric.test.2"},
		}
		actual := NewTriggerTargetMetrics(fetched)
		So(actual, ShouldHaveLength, 2)
		So(actual["metric.test.1"].Name, ShouldResemble, "metric.test.1")
		So(actual["metric.test.2"].Name, ShouldResemble, "metric.test.2")
	})
}

func TestTriggerTargetMetrics_Populate(t *testing.T) {
	type args struct {
		lastMetrics set[string]
		from        int64
		to          int64
	}
	tests := []struct {
		name string
		m    TriggerTargetMetrics
		args args
		want TriggerTargetMetrics
	}{
		{
			name: "origin do not have missing metrics",
			m: TriggerTargetMetrics{
				"metric.test.1": {Name: "metric.test.1", StartTime: 17, StopTime: 67, StepTime: 60, Values: []float64{0}},
				"metric.test.2": {Name: "metric.test.2", StartTime: 17, StopTime: 67, StepTime: 60, Values: []float64{0}},
			},
			args: args{
				lastMetrics: set[string]{
					"metric.test.1": void,
					"metric.test.2": void,
				},
				from: 17,
				to:   67,
			},
			want: TriggerTargetMetrics{
				"metric.test.1": {Name: "metric.test.1", StartTime: 17, StopTime: 67, StepTime: 60, Values: []float64{0}},
				"metric.test.2": {Name: "metric.test.2", StartTime: 17, StopTime: 67, StepTime: 60, Values: []float64{0}},
			},
		},
		{
			name: "origin have missing metrics",
			m: TriggerTargetMetrics{
				"metric.test.1": {Name: "metric.test.1", StartTime: 17, StopTime: 67, StepTime: 60, Values: []float64{0}},
			},
			args: args{
				lastMetrics: set[string]{
					"metric.test.1": void,
					"metric.test.2": void,
				},
				from: 17,
				to:   67,
			},
			want: TriggerTargetMetrics{
				"metric.test.1": {Name: "metric.test.1", StartTime: 17, StopTime: 67, StepTime: 60, Values: []float64{0}},
				"metric.test.2": {Name: "metric.test.2", StartTime: 17, StopTime: 67, StepTime: 60, Values: []float64{math.NaN()}},
			},
		},
	}
	Convey("Populate", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				actual := tt.m.Populate(tt.args.lastMetrics, tt.args.from, tt.args.to)
				So(actual, ShouldHaveLength, len(tt.want))
				for metricName, actualMetric := range actual {
					wantMetric, ok := tt.want[metricName]
					So(ok, ShouldBeTrue)
					So(actualMetric.StartTime, ShouldResemble, wantMetric.StartTime)
					So(actualMetric.StopTime, ShouldResemble, wantMetric.StopTime)
					So(actualMetric.StepTime, ShouldResemble, wantMetric.StepTime)
					So(actualMetric.Values, ShouldHaveLength, len(wantMetric.Values))
				}
			})
		}
	})
}
func TestNewTriggerMetricsWithCapacity(t *testing.T) {
	Convey("NewTriggerMetricsWithCapacity", t, func() {
		capacity := 10
		actual := NewTriggerMetricsWithCapacity(capacity)
		So(actual, ShouldNotBeNil)
		So(actual, ShouldHaveLength, 0)
	})
}

func TestTriggerMetrics_Populate(t *testing.T) {
	type args struct {
		lastCheck            map[string]moira.MetricState
		declaredAloneMetrics map[string]bool
		from                 int64
		to                   int64
	}
	tests := []struct {
		name           string
		triggerMetrics TriggerMetrics
		args           args
		want           TriggerMetrics
	}{
		{
			name: "origin do not have missing metrics",
			triggerMetrics: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
			},
			args: args{
				lastCheck: map[string]moira.MetricState{
					"metric.test.1": {Values: map[string]float64{"t1": 0}},
					"metric.test.2": {Values: map[string]float64{"t1": 0}},
				},
				declaredAloneMetrics: map[string]bool{},
				from:                 17,
				to:                   67,
			},
			want: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
			},
		},
		{
			name: "origin have missing metrics",
			triggerMetrics: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
				},
			},
			args: args{
				lastCheck: map[string]moira.MetricState{
					"metric.test.1": {Values: map[string]float64{"t1": 0}},
					"metric.test.2": {Values: map[string]float64{"t1": 0}},
				},
				declaredAloneMetrics: map[string]bool{},
				from:                 17,
				to:                   67,
			},
			want: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2", StartTime: 17, StopTime: 67, StepTime: 60, Values: []float64{math.NaN()}},
				},
			},
		},
		{
			name: "origin has no metrics for target",
			triggerMetrics: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
				"t2": TriggerTargetMetrics{},
			},
			args: args{
				lastCheck:            map[string]moira.MetricState{},
				declaredAloneMetrics: map[string]bool{},
				from:                 17,
				to:                   67,
			},
			want: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1", StopTime: 0},
					"metric.test.2": {Name: "metric.test.2", StopTime: 0},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1", StartTime: 17, StopTime: 67, StepTime: 60, Values: []float64{math.NaN()}},
					"metric.test.2": {Name: "metric.test.2", StartTime: 17, StopTime: 67, StepTime: 60, Values: []float64{math.NaN()}},
				},
			},
		},
	}
	Convey("Populate", t, func() {
		for _, testCase := range tests {
			Convey(testCase.name, func() {
				actual := testCase.triggerMetrics.Populate(
					testCase.args.lastCheck,
					testCase.args.declaredAloneMetrics,
					testCase.args.from,
					testCase.args.to,
				)

				So(actual, ShouldHaveLength, len(testCase.want))
				for targetName, metrics := range actual {
					wantMetrics, ok := testCase.want[targetName]
					So(metrics, ShouldHaveLength, len(wantMetrics))
					So(ok, ShouldBeTrue)
					for metricName, actualMetric := range metrics {
						wantMetric, ok := wantMetrics[metricName]
						So(ok, ShouldBeTrue)
						So(actualMetric.Name, ShouldResemble, wantMetric.Name)
						So(actualMetric.StartTime, ShouldResemble, wantMetric.StartTime)
						So(actualMetric.StopTime, ShouldResemble, wantMetric.StopTime)
						So(actualMetric.StepTime, ShouldResemble, wantMetric.StepTime)
						So(actualMetric.Values, ShouldHaveLength, len(wantMetric.Values))
					}
				}
			})
		}
	})
}

func TestTriggerMetrics_FilterAloneMetrics(t *testing.T) {
	Convey("FilterAloneMetrics", t, func() {
		Convey("origin does not have alone metrics", func() {
			m := TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
			}
			declared := map[string]bool{}
			filtered, alone, err := m.FilterAloneMetrics(declared)
			So(filtered, ShouldResemble, TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
			})
			So(alone, ShouldBeEmpty)
			So(err, ShouldBeNil)
		})
		Convey("origin has alone metrics", func() {
			m := TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.3": {Name: "metric.test.3"},
				},
			}
			declared := map[string]bool{"t2": true}
			filtered, alone, err := m.FilterAloneMetrics(declared)
			So(filtered, ShouldResemble, TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
			})
			So(alone, ShouldResemble, AloneMetrics{"t2": {Name: "metric.test.3"}})
			So(err, ShouldBeNil)
		})
		Convey("origin has alone metrics but it is not declared", func() {
			m := TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.3": {Name: "metric.test.3"},
				},
			}
			declared := map[string]bool{}
			filtered, alone, err := m.FilterAloneMetrics(declared)
			So(filtered, ShouldResemble, TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.3": {Name: "metric.test.3"},
				},
			})
			So(alone, ShouldBeEmpty)
			So(err, ShouldBeNil)
		})
		Convey("origin has targets that declared as alone metrics but it contains multiple metrics", func() {
			m := TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.3": {Name: "metric.test.3"},
					"metric.test.4": {Name: "metric.test.4"},
				},
			}
			declared := map[string]bool{"t2": true}
			filtered, alone, err := m.FilterAloneMetrics(declared)
			So(filtered, ShouldBeEmpty)
			So(alone, ShouldBeEmpty)
			So(err, ShouldBeError)
			So(err.(ErrUnexpectedAloneMetric).unexpected, ShouldContainKey, "t2")
			So(err.(ErrUnexpectedAloneMetric).unexpected["t2"], ShouldHaveLength, 2)
			So(err.(ErrUnexpectedAloneMetric).unexpected["t2"], ShouldContain, "metric.test.3")
			So(err.(ErrUnexpectedAloneMetric).unexpected["t2"], ShouldContain, "metric.test.4")
		})
		Convey("origin has targets that declared as alone metrics but it is empty", func() {
			m := TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
				"t2": TriggerTargetMetrics{},
			}
			declared := map[string]bool{"t2": true}
			filtered, alone, err := m.FilterAloneMetrics(declared)
			So(filtered, ShouldResemble, TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
			})
			So(alone, ShouldBeEmpty)
			So(err, ShouldBeNil)
		})
	})
}

func TestTriggerMetrics_Diff(t *testing.T) {
	tests := []struct {
		name                 string
		triggerMetrics       TriggerMetrics
		declaredAloneMetrics map[string]bool
		want                 map[string]set[string]
	}{
		{
			name: "all targets have same metrics",
			triggerMetrics: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
					"metric.test.3": {Name: "metric.test.3"},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
					"metric.test.3": {Name: "metric.test.3"},
				},
			},
			declaredAloneMetrics: map[string]bool{},
			want:                 map[string]set[string]{},
		},
		{
			name: "one target have missed metric",
			triggerMetrics: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
					"metric.test.3": {Name: "metric.test.3"},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
			},
			declaredAloneMetrics: map[string]bool{},
			want:                 map[string]set[string]{"t2": {"metric.test.3": void}},
		},
		{
			name: "one target is alone metric",
			triggerMetrics: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
					"metric.test.3": {Name: "metric.test.3"},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
				},
			},
			declaredAloneMetrics: map[string]bool{"t2": true},
			want:                 map[string]set[string]{},
		},
		{
			name: "another target have missed metric",
			triggerMetrics: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
					"metric.test.3": {Name: "metric.test.3"},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
					"metric.test.3": {Name: "metric.test.3"},
					"metric.test.4": {Name: "metric.test.4"},
				},
			},
			declaredAloneMetrics: map[string]bool{},
			want:                 map[string]set[string]{"t1": {"metric.test.4": void}},
		},
		{
			name: "one target is empty",
			triggerMetrics: TriggerMetrics{
				"t1": TriggerTargetMetrics{},
				"t2": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
					"metric.test.3": {Name: "metric.test.3"},
					"metric.test.4": {Name: "metric.test.4"},
				},
			},
			declaredAloneMetrics: map[string]bool{},
			want: map[string]set[string]{"t1": {
				"metric.test.1": void,
				"metric.test.2": void,
				"metric.test.3": void,
				"metric.test.4": void,
			}},
		},
		{
			name: "Multiple targets with different metrics",
			triggerMetrics: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.2": {Name: "metric.test.2"},
					"metric.test.3": {Name: "metric.test.3"},
					"metric.test.4": {Name: "metric.test.4"},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.3": {Name: "metric.test.3"},
					"metric.test.4": {Name: "metric.test.4"},
				},
				"t3": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
					"metric.test.4": {Name: "metric.test.4"},
				},
				"t4": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
					"metric.test.3": {Name: "metric.test.3"},
				},
			},
			declaredAloneMetrics: map[string]bool{},
			want: map[string]set[string]{
				"t1": {
					"metric.test.1": void,
				},
				"t2": {
					"metric.test.2": void,
				},
				"t3": {
					"metric.test.3": void,
				},
				"t4": {
					"metric.test.4": void,
				},
			},
		},
	}
	Convey("Diff", t, func() {
		for _, testCase := range tests {
			Convey(testCase.name, func() {
				actual := testCase.triggerMetrics.FindMissingMetrics(NewSet(testCase.declaredAloneMetrics))
				So(actual, ShouldResemble, testCase.want)
			})
		}
	})
}

func TestTriggerMetrics_ConvertForCheck(t *testing.T) {
	tests := []struct {
		name string
		m    TriggerMetrics
		want map[string]map[string]metricSource.MetricData
	}{
		{
			name: "origin is empty",
			m:    TriggerMetrics{},
			want: map[string]map[string]metricSource.MetricData{},
		},
		{
			name: "origin have metrics",
			m: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
			},
			want: map[string]map[string]metricSource.MetricData{
				"metric.test.1": {
					"t1": {Name: "metric.test.1"},
				},
				"metric.test.2": {
					"t1": {Name: "metric.test.2"},
				},
			},
		},
	}
	Convey("ConvertForCheck", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				actual := tt.m.ConvertForCheck()
				if actual == nil {
					t.Log("actual is nil")
				}
				So(actual, ShouldResemble, tt.want)
			})
		}
	})
}
