package notifier

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	mockMetricSource "github.com/moira-alert/moira/mock/metric_source"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
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
				expectedTo = roundToRetention(expectedTo)
				So(err, ShouldBeNil)
				from, to := resolveMetricsWindow(logger, trigger, pkg)
				So(from, ShouldEqual, expectedTo-timeRange+timeShift)
				So(to, ShouldEqual, expectedTo+timeShift)
			}
		})
		Convey("Window is not realtime: force realtime window", func() {
			pkg = oldTriggerEvents
			_, _, err := pkg.GetWindow()
			So(err, ShouldBeNil)
			from, to := resolveMetricsWindow(logger, trigger, pkg)
			So(from, ShouldEqual, roundToRetention(testLaunchTime.Add(-defaultTimeRange).UTC().Unix()))
			So(to, ShouldEqual, roundToRetention(testLaunchTime.UTC().Unix()))
		})
	})
	Convey("REMOTE TRIGGER | Resolve remote trigger metrics window", t, func() {
		trigger = remoteTrigger
		Convey("Window is wide: use package window to fetch limited historical data from graphite", func() {
			pkg = oldTriggerEvents
			expectedFrom, expectedTo, err := pkg.GetWindow()
			expectedFrom = roundToRetention(expectedFrom)
			expectedTo = roundToRetention(expectedTo)
			So(err, ShouldBeNil)
			from, to := resolveMetricsWindow(logger, trigger, pkg)
			So(from, ShouldEqual, expectedFrom)
			So(to, ShouldEqual, expectedTo)
		})
		Convey("Window is not wide: use shifted window to fetch extended historical data from graphite", func() {
			pkgs = []NotificationPackage{triggerJustCreatedEvents, realtimeTriggerEvents}
			for _, pkg := range pkgs {
				_, expectedTo, err := pkg.GetWindow()
				expectedTo = roundToRetention(expectedTo)
				So(err, ShouldBeNil)
				from, to := resolveMetricsWindow(logger, trigger, pkg)
				So(from, ShouldEqual, expectedTo-timeRange+timeShift)
				So(to, ShouldEqual, expectedTo+timeShift)
			}
		})
	})
	Convey("ANY TRIGGER | Zero time range, force default time range", t, func() {
		allTriggers := []moira.TriggerData{localTrigger, remoteTrigger}
		for _, trigger := range allTriggers {
			pkg := emptyEventsPackage
			from, to := resolveMetricsWindow(logger, trigger, pkg)
			expectedFrom := roundToRetention(testLaunchTime.Add(-defaultTimeRange).Unix())
			expectedTo := roundToRetention(testLaunchTime.Unix())
			_, _, err := pkg.GetWindow()
			So(err, ShouldResemble, fmt.Errorf("not enough data to resolve package window"))
			So(from, ShouldEqual, expectedFrom)
			So(to, ShouldEqual, expectedTo)
		}
	})
}

// TestGetMetricDataToShow tests to limited metricsData returns only necessary metricsData
func TestGetMetricDataToShow(t *testing.T) {
	givenSeries := map[string][]metricSource.MetricData{
		"t1": []metricSource.MetricData{
			*metricSource.MakeMetricData("metricPrefix.metricName1", []float64{1}, 1, 1),
			*metricSource.MakeMetricData("metricPrefix.metricName2", []float64{2}, 2, 2),
			*metricSource.MakeMetricData("metricPrefix.metricName3", []float64{3}, 3, 3),
		},
	}
	Convey("Limit series by non-empty whitelist", t, func() {
		Convey("MetricsData has necessary series", func() {
			metricsWhiteList := []string{"metricPrefix.metricName1", "metricPrefix.metricName2"}
			metricsData := getMetricDataToShow(givenSeries, metricsWhiteList)
			So(len(metricsData["t1"]), ShouldEqual, len(metricsWhiteList))
			So(metricsData["t1"][0].Name, ShouldEqual, metricsWhiteList[0])
			So(metricsData["t1"][1].Name, ShouldEqual, metricsWhiteList[1])
		})
		Convey("MetricsData has no necessary series", func() {
			metricsWhiteList := []string{"metricPrefix.metricName4"}
			metricsData := getMetricDataToShow(givenSeries, metricsWhiteList)
			So(len(metricsData["t1"]), ShouldEqual, 0)
		})
		Convey("MetricsData has necessary series and alone metrics target", func() {
			metricsWhiteList := []string{"metricPrefix.metricName1", "metricPrefix.metricName2"}
			givenSeries["t2"] = []metricSource.MetricData{
				*metricSource.MakeMetricData("metricPrefix.metricName4", []float64{1}, 1, 1),
			}
			metricsData := getMetricDataToShow(givenSeries, metricsWhiteList)
			So(len(metricsData["t1"]), ShouldEqual, 2)
			So(len(metricsData["t2"]), ShouldEqual, 1)
		})

	})
	Convey("Limit series by an empty whitelist", t, func() {
		metricsWhiteList := make([]string, 0)
		metricsData := getMetricDataToShow(givenSeries, metricsWhiteList)
		for metricDataInd := range metricsData["t1"] {
			So(metricsData["t1"][metricDataInd].Name, ShouldEqual, givenSeries["t1"][metricDataInd].Name)
		}
		So(len(metricsData), ShouldEqual, len(givenSeries))
	})
}

func TestFetchAvailableSeries(t *testing.T) {
	const (
		target = "testTarget"
		from   = 17
		to     = 67
	)
	Convey("Run fetchAvailableSeries", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		source := mockMetricSource.NewMockMetricSource(mockCtrl)
		result := mockMetricSource.NewMockFetchResult(mockCtrl)

		Convey("without errors", func() {
			gomock.InOrder(
				source.EXPECT().Fetch("testTarget", int64(17), int64(67), true).Return(result, nil).Times(1),
				result.EXPECT().GetMetricsData().Return(nil).Times(1),
			)
			_, err := fetchAvailableSeries(source, target, from, to)
			So(err, ShouldBeNil)
		})

		Convey("with error ErrEvaluateTargetFailedWithPanic", func() {
			var err error = local.ErrEvaluateTargetFailedWithPanic{}
			gomock.InOrder(
				source.EXPECT().Fetch("testTarget", int64(17), int64(67), true).Return(nil, err).Times(1),
				source.EXPECT().Fetch("testTarget", int64(17), int64(67), false).Return(result, nil).Times(1),
				result.EXPECT().GetMetricsData().Return(nil).Times(1),
			)
			_, err = fetchAvailableSeries(source, target, from, to)
			So(err, ShouldBeNil)
		})

		Convey("with error ErrEvaluateTargetFailedWithPanic and error again", func() {
			var err error = local.ErrEvaluateTargetFailedWithPanic{}
			var secondErr error = errors.New("Test error")
			gomock.InOrder(
				source.EXPECT().Fetch("testTarget", int64(17), int64(67), true).Return(nil, err).Times(1),
				source.EXPECT().Fetch("testTarget", int64(17), int64(67), false).Return(nil, secondErr).Times(1),
			)
			_, err = fetchAvailableSeries(source, target, from, to)
			So(err, ShouldNotBeNil)
		})

		Convey("with unknown error", func() {
			var err error = errors.New("Test error")
			gomock.InOrder(
				source.EXPECT().Fetch("testTarget", int64(17), int64(67), true).Return(nil, err).Times(1),
			)
			_, err = fetchAvailableSeries(source, target, from, to)
			So(err, ShouldNotBeNil)
		})
	})
}
