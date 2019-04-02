package notifier

import (
	"fmt"
	"testing"
	"time"

	"github.com/moira-alert/moira/metric_source"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
)

func TestResolveMetricsWindow(t *testing.T) {
	testLaunchTime := time.Now().UTC()
	logger, _ := logging.GetLogger("Notifier")
	emptyEventsPackage := NotificationPackage{}
	triggerJustCreatedEvents := NotificationPackage{
		Events: []moira.NotificationEvent{
			{Timestamp: testLaunchTime.Unix()},
		},
	}
	realtimeTriggerEvents := NotificationPackage{
		Events: []moira.NotificationEvent{
			{Timestamp: testLaunchTime.Add(-time.Minute).Unix()},
			{Timestamp: testLaunchTime.Unix()},
		},
	}
	oldTriggerEvents := NotificationPackage{
		Events: []moira.NotificationEvent{
			{Timestamp: testLaunchTime.Add(-720 * time.Hour).Unix()},
			{Timestamp: testLaunchTime.Add(-360 * time.Hour).Unix()},
		},
	}
	localTrigger := moira.TriggerData{ID: "redisTrigger", IsRemote: false}
	remoteTrigger := moira.TriggerData{ID: "remoteTrigger", IsRemote: true}
	timeRange := time.Unix(int64(defaultTimeRange.Seconds()), 0).Unix()
	timeShift := time.Unix(int64(defaultTimeShift.Seconds()), 0).Unix()
	var pkg NotificationPackage
	var pkgs []NotificationPackage
	var trigger moira.TriggerData
	Convey("LOCAL TRIGGER | Resolve trigger metrics window", t, func() {
		trigger = localTrigger
		Convey("Window is realtime: use shifted window to fetch actual data from redis", func() {
			pkgs = []NotificationPackage{triggerJustCreatedEvents, realtimeTriggerEvents}
			for _, pkg := range pkgs {
				_, expectedTo, err := pkg.GetWindow()
				So(err, ShouldBeNil)
				from, to := resolveMetricsWindow(logger, trigger, pkg)
				So(from, ShouldEqual, alignToMinutes(expectedTo)-timeRange+timeShift)
				So(to, ShouldEqual, expectedTo+timeShift)
			}
		})
		Convey("Window is not realtime: force realtime window", func() {
			pkg = oldTriggerEvents
			_, _, err := pkg.GetWindow()
			So(err, ShouldBeNil)
			from, to := resolveMetricsWindow(logger, trigger, pkg)
			So(from, ShouldEqual, alignToMinutes(testLaunchTime.Add(-defaultTimeRange).UTC().Unix()))
			So(to, ShouldEqual, testLaunchTime.UTC().Unix())
		})
	})
	Convey("REMOTE TRIGGER | Resolve remote trigger metrics window", t, func() {
		trigger = remoteTrigger
		Convey("Window is wide: use package window to fetch limited historical data from graphite", func() {
			pkg = oldTriggerEvents
			expectedFrom, expectedTo, err := pkg.GetWindow()
			So(err, ShouldBeNil)
			from, to := resolveMetricsWindow(logger, trigger, pkg)
			So(from, ShouldEqual, alignToMinutes(expectedFrom))
			So(to, ShouldEqual, expectedTo)
		})
		Convey("Window is not wide: use shifted window to fetch extended historical data from graphite", func() {
			pkgs = []NotificationPackage{triggerJustCreatedEvents, realtimeTriggerEvents}
			for _, pkg := range pkgs {
				_, expectedTo, err := pkg.GetWindow()
				So(err, ShouldBeNil)
				from, to := resolveMetricsWindow(logger, trigger, pkg)
				So(from, ShouldEqual, alignToMinutes(expectedTo-timeRange+timeShift))
				So(to, ShouldEqual, expectedTo+timeShift)
			}
		})
	})
	Convey("ANY TRIGGER | Zero time range, force default time range", t, func() {
		allTriggers := []moira.TriggerData{localTrigger, remoteTrigger}
		for _, trigger := range allTriggers {
			pkg := emptyEventsPackage
			from, to := resolveMetricsWindow(logger, trigger, pkg)
			expectedFrom := testLaunchTime.Add(-defaultTimeRange).Unix()
			expectedTo := testLaunchTime.Unix()
			_, _, err := pkg.GetWindow()
			So(err, ShouldResemble, fmt.Errorf("not enough data to resolve package window"))
			So(from, ShouldEqual, alignToMinutes(expectedFrom))
			So(to, ShouldEqual, expectedTo)
		}
	})
}

// TestGetMetricDataToShow tests to limited metricsData returns only necessary metricsData
func TestGetMetricDataToShow(t *testing.T) {
	givenSeries := []*metricSource.MetricData{
		metricSource.MakeMetricData("metricPrefix.metricName1", []float64{1}, 1, 1),
		metricSource.MakeMetricData("metricPrefix.metricName2", []float64{2}, 2, 2),
		metricSource.MakeMetricData("metricPrefix.metricName3", []float64{3}, 3, 3),
	}
	Convey("Limit series by non-empty whitelist", t, func() {
		Convey("MetricsData has necessary series", func() {
			metricsWhiteList := []string{"metricPrefix.metricName1", "metricPrefix.metricName2"}
			metricsData := getMetricDataToShow(givenSeries, metricsWhiteList)
			So(len(metricsData), ShouldEqual, len(metricsWhiteList))
			So(metricsData[0].Name, ShouldEqual, metricsWhiteList[0])
			So(metricsData[1].Name, ShouldEqual, metricsWhiteList[1])
		})
		Convey("MetricsData has no necessary series", func() {
			metricsWhiteList := []string{"metricPrefix.metricName4"}
			metricsData := getMetricDataToShow(givenSeries, metricsWhiteList)
			So(len(metricsData), ShouldEqual, 0)
		})
	})
	Convey("Limit series by an empty whitelist", t, func() {
		metricsWhiteList := make([]string, 0)
		metricsData := getMetricDataToShow(givenSeries, metricsWhiteList)
		for metricDataInd := range metricsData {
			So(metricsData[metricDataInd].Name, ShouldEqual, givenSeries[metricDataInd].Name)
		}
		So(len(metricsData), ShouldEqual, len(givenSeries))
	})
}
