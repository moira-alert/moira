package metricSource

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMakeEmptyTriggerMetricsData(t *testing.T) {
	Convey("Just make empty TriggerMetricsData", t, func(c C) {
		c.So(*(MakeEmptyTriggerMetricsData()), ShouldResemble, TriggerMetricsData{
			Main:       make([]*MetricData, 0),
			Additional: make([]*MetricData, 0),
		})
	})
}

func TestMakeTriggerMetricsData(t *testing.T) {
	Convey("Just make empty TriggerMetricsData", t, func(c C) {
		c.So(*(MakeTriggerMetricsData(make([]*MetricData, 0), make([]*MetricData, 0))), ShouldResemble, TriggerMetricsData{
			Main:       make([]*MetricData, 0),
			Additional: make([]*MetricData, 0),
		})
	})

	Convey("Just make TriggerMetricsData only with main", t, func(c C) {
		c.So(*(MakeTriggerMetricsData([]*MetricData{MakeMetricData("000", make([]float64, 0), 10, 0)}, make([]*MetricData, 0))), ShouldResemble, TriggerMetricsData{
			Main:       []*MetricData{MakeMetricData("000", make([]float64, 0), 10, 0)},
			Additional: make([]*MetricData, 0),
		})
	})

	Convey("Just make TriggerMetricsData with main and additional", t, func(c C) {
		c.So(*(MakeTriggerMetricsData([]*MetricData{MakeMetricData("000", make([]float64, 0), 10, 0)}, []*MetricData{MakeMetricData("000", make([]float64, 0), 10, 0)})), ShouldResemble, TriggerMetricsData{
			Main:       []*MetricData{MakeMetricData("000", make([]float64, 0), 10, 0)},
			Additional: []*MetricData{MakeMetricData("000", make([]float64, 0), 10, 0)},
		})
	})
}

func TestGetTargetName(t *testing.T) {
	tts := TriggerMetricsData{}

	Convey("GetMainTargetName", t, func(c C) {
		c.So(tts.GetMainTargetName(), ShouldResemble, "t1")
	})

	Convey("GetAdditionalTargetName", t, func(c C) {
		for i := 0; i < 5; i++ {
			c.So(tts.GetAdditionalTargetName(i), ShouldResemble, fmt.Sprintf("t%v", i+2))
		}
	})
}

func TestTriggerTimeSeriesHasOnlyWildcards(t *testing.T) {
	Convey("Main metrics data has wildcards only", t, func(c C) {
		tts := TriggerMetricsData{
			Main: []*MetricData{{Wildcard: true}},
		}
		c.So(tts.HasOnlyWildcards(), ShouldBeTrue)

		tts1 := TriggerMetricsData{
			Main: []*MetricData{{Wildcard: true}, {Wildcard: true}},
		}
		c.So(tts1.HasOnlyWildcards(), ShouldBeTrue)
	})

	Convey("Main metrics data has not only wildcards", t, func(c C) {
		tts := TriggerMetricsData{
			Main: []*MetricData{{Wildcard: false}},
		}
		c.So(tts.HasOnlyWildcards(), ShouldBeFalse)

		tts1 := TriggerMetricsData{
			Main: []*MetricData{{Wildcard: false}, {Wildcard: true}},
		}
		c.So(tts1.HasOnlyWildcards(), ShouldBeFalse)

		tts2 := TriggerMetricsData{
			Main: []*MetricData{{Wildcard: false}, {Wildcard: false}},
		}
		c.So(tts2.HasOnlyWildcards(), ShouldBeFalse)
	})

	Convey("Additional metrics data has wildcards but Main not", t, func(c C) {
		tts := TriggerMetricsData{
			Main:       []*MetricData{{Wildcard: false}},
			Additional: []*MetricData{{Wildcard: true}, {Wildcard: true}},
		}
		c.So(tts.HasOnlyWildcards(), ShouldBeFalse)
	})
}
