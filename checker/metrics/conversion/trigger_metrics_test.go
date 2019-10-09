package conversion

import (
	"math"
	"reflect"
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
		lastMetrics map[string]bool
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
				lastMetrics: map[string]bool{
					"metric.test.1": true,
					"metric.test.2": true,
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
				lastMetrics: map[string]bool{
					"metric.test.1": true,
					"metric.test.2": true,
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
		lastCheck moira.CheckData
		from      int64
		to        int64
	}
	tests := []struct {
		name string
		m    TriggerMetrics
		args args
		want TriggerMetrics
	}{
		{
			name: "origin do not have missing metrics",
			m: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
			},
			args: args{
				lastCheck: moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"metric.test.1": {Values: map[string]float64{"t1": 0}},
						"metric.test.2": {Values: map[string]float64{"t1": 0}},
					},
				},
				from: 17,
				to:   67,
			},
			want: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
			},
		},
		{
			name: "origin have missing alone metric",
			m: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
			},
			args: args{
				lastCheck: moira.CheckData{
					MetricsToTargetRelation: map[string]string{"t2": "metric.test.3"},
					Metrics: map[string]moira.MetricState{
						"metric.test.1": {Values: map[string]float64{"t1": 0, "t2": 0}},
						"metric.test.2": {Values: map[string]float64{"t1": 0, "t2": 0}},
					},
				},
				from: 17,
				to:   67,
			},
			want: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
				"t2": {
					"metric.test.3": {Name: "metric.test.3", StartTime: 17, StopTime: 67, StepTime: 60, Values: []float64{math.NaN()}},
				},
			},
		},
		{
			name: "origin have missing metrics",
			m: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
				},
			},
			args: args{
				lastCheck: moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"metric.test.1": {Values: map[string]float64{"t1": 0}},
						"metric.test.2": {Values: map[string]float64{"t1": 0}},
					},
				},
				from: 17,
				to:   67,
			},
			want: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2", StartTime: 17, StopTime: 67, StepTime: 60, Values: []float64{math.NaN()}},
				},
			},
		},
		{
			name: "origin have missing metrics and alone metrics",
			m: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.3": {Name: "metric.test.3"},
				},
			},
			args: args{
				lastCheck: moira.CheckData{
					MetricsToTargetRelation: map[string]string{"t2": "metric.test.3"},
					Metrics: map[string]moira.MetricState{
						"metric.test.1": {Values: map[string]float64{"t1": 0, "t2": 0}},
						"metric.test.2": {Values: map[string]float64{"t1": 0, "t2": 0}},
					},
				},
				from: 17,
				to:   67,
			},
			want: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2", StartTime: 17, StopTime: 67, StepTime: 60, Values: []float64{math.NaN()}},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.3": {Name: "metric.test.3"},
				},
			},
		},
		{
			name: "origin have target with missing metrics and alone metrics",
			m: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
					"metric.test.3": {Name: "metric.test.3"},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.4": {Name: "metric.test.4"},
				},
				"t3": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
			},
			args: args{
				lastCheck: moira.CheckData{
					MetricsToTargetRelation: map[string]string{"t2": "metric.test.4"},
					Metrics: map[string]moira.MetricState{
						"metric.test.1": {Values: map[string]float64{"t1": 0, "t2": 0, "t3": 0}},
						"metric.test.2": {Values: map[string]float64{"t1": 0, "t2": 0, "t3": 0}},
						"metric.test.3": {Values: map[string]float64{"t1": 0, "t2": 0, "t3": 0}},
						"metric.test.4": {Values: map[string]float64{"t1": 0, "t2": 0, "t3": 0}},
					},
				},
				from: 17,
				to:   67,
			},
			want: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
					"metric.test.3": {Name: "metric.test.3"},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.4": {Name: "metric.test.4"},
				},
				"t3": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
					"metric.test.3": {Name: "metric.test.3", StartTime: 17, StopTime: 67, StepTime: 60, Values: []float64{math.NaN()}},
				},
			},
		},
	}
	Convey("Populate", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				actual := tt.m.Populate(tt.args.lastCheck, tt.args.from, tt.args.to)
				So(actual, ShouldHaveLength, len(tt.want))
				for targetName, metrics := range actual {
					wantMetrics, ok := tt.want[targetName]
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
	tests := []struct {
		name  string
		m     TriggerMetrics
		want  TriggerMetrics
		want1 map[string]metricSource.MetricData
	}{
		{
			name: "origin does not have alone metrics",
			m: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
			},
			want: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
			},
			want1: map[string]metricSource.MetricData{},
		},
		{
			name: "origin has alone metrics",
			m: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.3": {Name: "metric.test.3"},
				},
			},
			want: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
			},
			want1: map[string]metricSource.MetricData{"t2": {Name: "metric.test.3"}},
		},
	}
	Convey("FilterAloneMetrics", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				filtered, alone := tt.m.FilterAloneMetrics()
				So(filtered, ShouldResemble, tt.want)
				So(alone, ShouldResemble, tt.want1)
			})
		}
	})
}

func TestTriggerMetrics_Diff(t *testing.T) {
	tests := []struct {
		name string
		m    TriggerMetrics
		want map[string]map[string]bool
	}{
		{
			name: "all targets have same metrics",
			m: TriggerMetrics{
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
			want: map[string]map[string]bool{},
		},
		{
			name: "one target have missed metric",
			m: TriggerMetrics{
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
			want: map[string]map[string]bool{"t2": {"metric.test.3": true}},
		},
		{
			name: "one target is alone metric",
			m: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
					"metric.test.3": {Name: "metric.test.3"},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
				},
			},
			want: map[string]map[string]bool{},
		},
		{
			name: "another target have missed metric",
			m: TriggerMetrics{
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
			want: map[string]map[string]bool{"t1": {"metric.test.4": true}},
		},
		{
			name: "one target is empty",
			m: TriggerMetrics{
				"t1": TriggerTargetMetrics{},
				"t2": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
					"metric.test.3": {Name: "metric.test.3"},
					"metric.test.4": {Name: "metric.test.4"},
				},
			},
			want: map[string]map[string]bool{"t1": {
				"metric.test.1": true,
				"metric.test.2": true,
				"metric.test.3": true,
				"metric.test.4": true,
			}},
		},
		{
			name: "Multiple targets with different metrics",
			m: TriggerMetrics{
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
			want: map[string]map[string]bool{
				"t1": {
					"metric.test.1": true,
				},
				"t2": {
					"metric.test.2": true,
				},
				"t3": {
					"metric.test.3": true,
				},
				"t4": {
					"metric.test.4": true,
				},
			},
		},
	}
	Convey("Diff", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				actual := tt.m.Diff()
				So(actual, ShouldResemble, tt.want)
			})
		}
	})
}

func TestTriggerMetrics_multiMetricsTarget(t *testing.T) {
	tests := []struct {
		name  string
		m     TriggerMetrics
		want  string
		want1 setHelper
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.m.multiMetricsTarget()
			if got != tt.want {
				t.Errorf("TriggerMetrics.multiMetricsTarget() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("TriggerMetrics.multiMetricsTarget() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
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
				"metric.test.1": map[string]metricSource.MetricData{
					"t1": {Name: "metric.test.1"},
				},
				"metric.test.2": map[string]metricSource.MetricData{
					"t1": {Name: "metric.test.2"},
				},
			},
		},
		{
			name: "origin have metrics and target with empty metrics",
			m: TriggerMetrics{
				"t1": TriggerTargetMetrics{
					"metric.test.1": {Name: "metric.test.1"},
					"metric.test.2": {Name: "metric.test.2"},
				},
				"t2": TriggerTargetMetrics{
					"metric.test.3": {Name: "metric.test.3"},
				},
			},
			want: map[string]map[string]metricSource.MetricData{
				"metric.test.1": map[string]metricSource.MetricData{
					"t1": {Name: "metric.test.1"},
					"t2": {Name: "metric.test.3"},
				},
				"metric.test.2": map[string]metricSource.MetricData{
					"t1": {Name: "metric.test.2"},
					"t2": {Name: "metric.test.3"},
				},
			},
		},
	}
	Convey("ConvertForCheck", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				actual := tt.m.ConvertForCheck()
				So(actual, ShouldResemble, tt.want)
			})
		}
	})
}
