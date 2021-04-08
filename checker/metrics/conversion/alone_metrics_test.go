package conversion

import (
	"testing"

	metricsource "github.com/moira-alert/moira/metric_source"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewAloneMetricsWithCapacity(t *testing.T) {
	Convey("NewAloneMetricsWithCapacity", t, func() {
		actual := NewAloneMetricsWithCapacity(10)
		So(actual, ShouldResemble, AloneMetrics{})
	})
}

func TestAloneMetrics_Populate(t *testing.T) {
	Convey("Populate alone metrics", t, func() {
		Convey("Missing one metric", func() {
			m := AloneMetrics{"t1": metricsource.MetricData{Name: "metric.test.1", StepTime: 60, StartTime: 0, StopTime: 3600}}
			lastCheckMetricsToTargetRelation := map[string]string{
				"t1": "metric.test.1",
				"t2": "metric.test.2",
			}
			declaredAloneMetrics := map[string]bool{
				"t1": true,
				"t2": true,
			}
			const from = 0
			const to = 3600
			populated, err := m.Populate(lastCheckMetricsToTargetRelation, declaredAloneMetrics, from, to)
			So(err, ShouldBeNil)
			// We assume alone metrics to be like this
			// AloneMetrics {
			// 	"t1": metricsource.MetricData{Name: "metric.test.1", StepTime: 60, StartTime: 0, StopTime: 3600},
			// 	"t2": metricsource.MetricData{Name: "metric.test.2", StepTime: 60, StartTime: 0, StopTime: 3600},
			// }
			So(populated, ShouldHaveLength, 2)
			So(populated, ShouldContainKey, "t1")
			So(populated, ShouldContainKey, "t2")
			// We cannot just use ShouldResemble as it not correctly works with NaN
			So(populated["t2"].StartTime, ShouldResemble, int64(0))
			So(populated["t2"].StopTime, ShouldResemble, int64(3600))
			So(populated["t2"].StepTime, ShouldResemble, int64(60))
			So(populated["t2"].Values, ShouldHaveLength, 60)
		})
		Convey("Missing one metric and no other metrics provided. Use default values", func() {
			m := AloneMetrics{}
			lastCheckMetricsToTargetRelation := map[string]string{
				"t1": "metric.test.1",
			}
			declaredAloneMetrics := map[string]bool{
				"t1": true,
			}
			const from = 0
			const to = 3600
			populated, err := m.Populate(lastCheckMetricsToTargetRelation, declaredAloneMetrics, from, to)
			So(err, ShouldBeNil)
			// We assume alone metrics to be like this
			// AloneMetrics {
			// 	"t1": metricsource.MetricData{Name: "metric.test.1", StepTime: 60, StartTime: 0, StopTime: 3600},
			// }
			So(populated, ShouldHaveLength, 1)
			So(populated, ShouldContainKey, "t1")
			// We cannot just use ShouldResemble as it not correctly works with NaN
			So(populated["t1"].StartTime, ShouldResemble, int64(0))
			So(populated["t1"].StopTime, ShouldResemble, int64(3600))
			So(populated["t1"].StepTime, ShouldResemble, int64(60))
			So(populated["t1"].Values, ShouldHaveLength, 60)
		})
		Convey("One declared alone metrics target do not have metrics and metrics in last check", func() {
			m := AloneMetrics{}
			lastCheckMetricsToTargetRelation := map[string]string{}
			declaredAloneMetrics := map[string]bool{
				"t1": true,
			}
			const from = 0
			const to = 3600
			populated, err := m.Populate(lastCheckMetricsToTargetRelation, declaredAloneMetrics, from, to)
			So(err, ShouldResemble, ErrEmptyAloneMetricsTarget{targetName: "t1"})
			So(populated, ShouldResemble, AloneMetrics{})
		})
	})
}
