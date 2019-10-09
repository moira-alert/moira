package metricSource

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewTriggerMetricsData(t *testing.T) {
	Convey("Just make empty TriggerMetricsData", t, func() {
		So(*(NewTriggerMetricsData()), ShouldResemble, TriggerMetricsData{
			Main:       make([]*MetricData, 0),
			Additional: make([]*MetricData, 0),
		})
	})
}

func TestMakeTriggerMetricsData(t *testing.T) {
	Convey("Just make empty TriggerMetricsData", t, func() {
		So(*(MakeTriggerMetricsData(make([]*MetricData, 0), make([]*MetricData, 0))), ShouldResemble, TriggerMetricsData{
			Main:       make([]*MetricData, 0),
			Additional: make([]*MetricData, 0),
		})
	})

	Convey("Just make TriggerMetricsData only with main", t, func() {
		So(*(MakeTriggerMetricsData([]*MetricData{MakeMetricData("000", make([]float64, 0), 10, 0)}, make([]*MetricData, 0))), ShouldResemble, TriggerMetricsData{
			Main:       []*MetricData{MakeMetricData("000", make([]float64, 0), 10, 0)},
			Additional: make([]*MetricData, 0),
		})
	})

	Convey("Just make TriggerMetricsData with main and additional", t, func() {
		So(*(MakeTriggerMetricsData([]*MetricData{MakeMetricData("000", make([]float64, 0), 10, 0)}, []*MetricData{MakeMetricData("000", make([]float64, 0), 10, 0)})), ShouldResemble, TriggerMetricsData{
			Main:       []*MetricData{MakeMetricData("000", make([]float64, 0), 10, 0)},
			Additional: []*MetricData{MakeMetricData("000", make([]float64, 0), 10, 0)},
		})
	})
}

func TestGetTargetName(t *testing.T) {
	tts := TriggerMetricsData{}

	Convey("GetMainTargetName", t, func() {
		So(tts.GetMainTargetName(), ShouldResemble, "t1")
	})

	Convey("GetAdditionalTargetName", t, func() {
		for i := 0; i < 5; i++ {
			So(tts.GetAdditionalTargetName(i), ShouldResemble, fmt.Sprintf("t%v", i+2))
		}
	})
}

func TestTriggerTimeSeriesHasOnlyWildcards(t *testing.T) {
	Convey("Main metrics data has wildcards only", t, func() {
		tts := TriggerMetricsData{
			Main: []*MetricData{{Wildcard: true}},
		}
		So(tts.HasOnlyWildcards(), ShouldBeTrue)

		tts1 := TriggerMetricsData{
			Main: []*MetricData{{Wildcard: true}, {Wildcard: true}},
		}
		So(tts1.HasOnlyWildcards(), ShouldBeTrue)
	})

	Convey("Main metrics data has not only wildcards", t, func() {
		tts := TriggerMetricsData{
			Main: []*MetricData{{Wildcard: false}},
		}
		So(tts.HasOnlyWildcards(), ShouldBeFalse)

		tts1 := TriggerMetricsData{
			Main: []*MetricData{{Wildcard: false}, {Wildcard: true}},
		}
		So(tts1.HasOnlyWildcards(), ShouldBeFalse)

		tts2 := TriggerMetricsData{
			Main: []*MetricData{{Wildcard: false}, {Wildcard: false}},
		}
		So(tts2.HasOnlyWildcards(), ShouldBeFalse)
	})

	Convey("Additional metrics data has wildcards but Main not", t, func() {
		tts := TriggerMetricsData{
			Main:       []*MetricData{{Wildcard: false}},
			Additional: []*MetricData{{Wildcard: true}, {Wildcard: true}},
		}
		So(tts.HasOnlyWildcards(), ShouldBeFalse)
	})
}
