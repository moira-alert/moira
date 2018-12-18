package notifier

import (
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
			{Timestamp: time.Unix(1800, 0).Unix()},
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
	Convey("Resolve redis trigger metrics window", t, func() {
		trigger := moira.TriggerData{ID: "redisTrigger", IsRemote: false}
		defaultTimeRangePackages := make([]NotificationPackage, 0)
		defaultTimeRangePackages = append(defaultTimeRangePackages,
			emptyEventsPackage, triggerJustCreatedEvents, realtimeTriggerEvents, oldTriggerEvents)
		now := time.Now()
		expectedFrom := now.UTC().Add(-defaultTimeRange).Unix()
		expectedTo := now.UTC().Unix()
		for _, pkg := range defaultTimeRangePackages {
			from, to := resolveMetricsWindow(logger, trigger, pkg)
			So(from, ShouldEqual, expectedFrom)
			So(to, ShouldEqual, expectedTo)
		}
	})
	Convey("Resolve remote trigger metrics window", t, func() {
		trigger := moira.TriggerData{ID: "remoteTrigger", IsRemote: true}
		highTimeRangePackages := make([]NotificationPackage, 0)
		highTimeRangePackages = append(highTimeRangePackages, realtimeTriggerEvents, oldTriggerEvents)
		Convey("High time range, use package window", func() {
			for _, pkg := range highTimeRangePackages {
				from, to := resolveMetricsWindow(logger, trigger, pkg)
				expectedFrom, expectedTo, err := pkg.GetWindow()
				So(err, ShouldBeNil)
				So(from, ShouldEqual, expectedFrom)
				So(to, ShouldEqual, expectedTo)
			}
		})
		Convey("Low time range, takes to extend", func() {
			from, to := resolveMetricsWindow(logger, trigger, triggerJustCreatedEvents)
			So(from, ShouldEqual, 1800/2)
			So(to, ShouldEqual, 1800+1800/2)
		})
		Convey("Now time range, force default range", func() {
			from, to := resolveMetricsWindow(logger, trigger, emptyEventsPackage)
			now := time.Now()
			expectedFrom := now.UTC().Add(-defaultTimeRange).Unix()
			expectedTo := now.UTC().Unix()
			So(from, ShouldEqual, expectedFrom)
			So(to, ShouldEqual, expectedTo)
		})
	})
}
