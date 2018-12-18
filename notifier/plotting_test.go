package notifier

import (
	"fmt"
	"testing"
	"time"

	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
)

func TestResolveMetricsWindow(t *testing.T) {
	logger, _ := logging.GetLogger("Notifier")
	emptyEventsPackage := NotificationPackage{}
	triggerJustCreatedEvents := NotificationPackage{
		Events: []moira.NotificationEvent{
			{Timestamp: time.Unix(3600, 0).Unix()},
		},
	}
	realtimeTriggerEvents := NotificationPackage{
		Events: []moira.NotificationEvent{
			{Timestamp: time.Unix(1800, 0).Unix()},
			{Timestamp: time.Unix(0, 0).Unix()},
		},
	}
	oldTriggerEvents := NotificationPackage{
		Events: []moira.NotificationEvent{
			{Timestamp: time.Unix(0, 0).Unix()},
			{Timestamp: time.Unix(604800, 0).Unix()},
		},
	}
	redisTrigger := moira.TriggerData{ID: "redisTrigger", IsRemote: false}
	remoteTrigger := moira.TriggerData{ID: "remoteTrigger", IsRemote: true}
	timerange := time.Unix(int64(defaultTimeRange.Seconds()), 0).Unix()
	timeShift := time.Unix(int64(defaultTimeShift.Seconds()), 0).Unix()
	Convey("REDIS TRIGGER | Resolve trigger metrics window", t, func() {
		Convey("High time range, use package window", func() {
			defaultTimeRangePackages := make([]NotificationPackage, 0)
			defaultTimeRangePackages = append(defaultTimeRangePackages,
				triggerJustCreatedEvents, realtimeTriggerEvents, oldTriggerEvents)
			for _, pkg := range defaultTimeRangePackages {
				_, expectedTo, err := pkg.GetWindow()
				So(err, ShouldBeNil)
				from, to := resolveMetricsWindow(logger, redisTrigger, pkg)
				So(from, ShouldEqual, expectedTo-timerange+timeShift)
				So(to, ShouldEqual, expectedTo+timeShift)
			}
		})
	})
	Convey("REMOTE TRIGGER | Resolve remote trigger metrics window", t, func() {
		highTimeRangePackages := make([]NotificationPackage, 0)
		highTimeRangePackages = append(highTimeRangePackages, realtimeTriggerEvents, oldTriggerEvents)
		Convey("High time range, use package window", func() {
			for _, pkg := range highTimeRangePackages {
				from, to := resolveMetricsWindow(logger, remoteTrigger, pkg)
				expectedFrom, expectedTo, err := pkg.GetWindow()
				So(err, ShouldBeNil)
				So(from, ShouldEqual, expectedFrom)
				So(to, ShouldEqual, expectedTo)
				fmt.Println(expectedTo)
			}
		})
		Convey("Low time range, takes to extend", func() {
			pkg := triggerJustCreatedEvents
			from, to := resolveMetricsWindow(logger, remoteTrigger, pkg)
			_, expectedTo, err := pkg.GetWindow()
			So(err, ShouldBeNil)
			So(from, ShouldEqual, expectedTo-timerange+timeShift)
			So(to, ShouldEqual, expectedTo+timeShift)
		})
	})
	Convey("ANY TRIGGER | Zero time range, force default time range", t, func() {
		allTriggers := []moira.TriggerData{redisTrigger, remoteTrigger}
		for _, trigger := range allTriggers {
			pkg := emptyEventsPackage
			from, to := resolveMetricsWindow(logger, trigger, pkg)
			now := time.Now()
			expectedFrom := now.UTC().Add(-defaultTimeRange).Unix()
			expectedTo := now.UTC().Unix()
			_, _, err := pkg.GetWindow()
			So(err, ShouldResemble, fmt.Errorf("not enough data to resolve package window"))
			So(from, ShouldEqual, expectedFrom)
			So(to, ShouldEqual, expectedTo)
		}
	})
}
