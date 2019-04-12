package checker

import (
	"testing"

	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetMaintenanceInfo(t *testing.T) {
	Convey("Test getting right maintenance info from trigger or metric", t, func() {
		triggerStateWithRemovedMaintenance := moira.CheckData{Maintenance: 0}
		setMaintenance(&triggerStateWithRemovedMaintenance.MaintenanceInfo, "start trigger user", 100)
		removeMaintenance(&triggerStateWithRemovedMaintenance.MaintenanceInfo, "remove triggre user", 1000)

		metricStateWithRemovedMaintenance := moira.MetricState{Maintenance: 0}
		setMaintenance(&metricStateWithRemovedMaintenance.MaintenanceInfo, "start metric user", 100)
		removeMaintenance(&metricStateWithRemovedMaintenance.MaintenanceInfo, "remove metric user", 1000)

		Convey("Metric state is nil", func() {
			actualInfo, actualTS := getMaintenanceInfo(&triggerStateWithRemovedMaintenance, nil)
			So(actualInfo, ShouldResemble, triggerStateWithRemovedMaintenance.MaintenanceInfo)
			So(actualTS, ShouldResemble, triggerStateWithRemovedMaintenance.Maintenance)
		})

		Convey("Trigger state is nil", func() {
			actualInfo, actualTS := getMaintenanceInfo(nil, &metricStateWithRemovedMaintenance)
			So(actualInfo, ShouldResemble, metricStateWithRemovedMaintenance.MaintenanceInfo)
			So(actualTS, ShouldResemble, metricStateWithRemovedMaintenance.Maintenance)
		})

		Convey("Trigger and Metric state has data", func() {
			Convey("Trigger never be in maintenance but metric with maintenance", func() {
				triggerState := moira.CheckData{}
				Convey("in maintenance", func() {
					metricState := moira.MetricState{Maintenance: 1000}
					setMaintenance(&metricState.MaintenanceInfo, "user", 100)
					actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
					So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
					So(actualTS, ShouldResemble, metricState.Maintenance)
				})

				Convey("removed maintenance", func() {
					actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricStateWithRemovedMaintenance)
					So(actualInfo, ShouldResemble, metricStateWithRemovedMaintenance.MaintenanceInfo)
					So(actualTS, ShouldResemble, metricStateWithRemovedMaintenance.Maintenance)
				})
			})

			Convey("Metric never be in maintenance but trigger with maintenance", func() {
				metricState := moira.MetricState{}
				Convey("in maintenance", func() {
					triggerState := moira.CheckData{Maintenance: 1000}
					setMaintenance(&triggerState.MaintenanceInfo, "user", 100)
					actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
					So(actualInfo, ShouldResemble, triggerState.MaintenanceInfo)
					So(actualTS, ShouldResemble, triggerState.Maintenance)
				})

				Convey("removed maintenance", func() {
					actualInfo, actualTS := getMaintenanceInfo(&triggerStateWithRemovedMaintenance, &metricState)
					So(actualInfo, ShouldResemble, triggerStateWithRemovedMaintenance.MaintenanceInfo)
					So(actualTS, ShouldResemble, triggerStateWithRemovedMaintenance.Maintenance)
				})
			})

			Convey("Trigger and metric has maintenance", func() {
				triggerState := moira.CheckData{Maintenance: 1000}
				setMaintenance(&triggerState.MaintenanceInfo, "trigger user", 100)
				metricState := moira.MetricState{Maintenance: 1000}
				setMaintenance(&metricState.MaintenanceInfo, "metric user", 200)
				Convey("Both has set maintenance", func() {
					Convey("maintenance TS are equal", func() {
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						So(actualTS, ShouldResemble, metricState.Maintenance)
					})
					Convey("metric maintenance TS are more", func() {
						metricState.Maintenance = 2000
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						So(actualTS, ShouldResemble, metricState.Maintenance)
					})
					Convey("trigger maintenance TS are more", func() {
						triggerState.Maintenance = 2000
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						So(actualInfo, ShouldResemble, triggerState.MaintenanceInfo)
						So(actualTS, ShouldResemble, triggerState.Maintenance)
					})
				})

				Convey("Both has removed maintenance, compare remove time", func() {
					metricState.Maintenance = 0
					triggerState.Maintenance = 0
					Convey("remove TS are equal", func() {
						removeMaintenance(&metricState.MaintenanceInfo, "metric user", 1500)
						removeMaintenance(&triggerState.MaintenanceInfo, "trigger user", 1500)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						So(actualTS, ShouldResemble, metricState.Maintenance)
					})
					Convey("metric remove TS are more", func() {
						removeMaintenance(&metricState.MaintenanceInfo, "metric user", 1600)
						removeMaintenance(&triggerState.MaintenanceInfo, "trigger user", 1500)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						So(actualTS, ShouldResemble, metricState.Maintenance)
					})
					Convey("trigger remove TS are more", func() {
						removeMaintenance(&metricState.MaintenanceInfo, "metric user", 1400)
						removeMaintenance(&triggerState.MaintenanceInfo, "trigger user", 1500)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						So(actualInfo, ShouldResemble, triggerState.MaintenanceInfo)
						So(actualTS, ShouldResemble, triggerState.Maintenance)
					})
				})

				Convey("Trigger has removed maintenance metric has set maintenance", func() {
					triggerState.Maintenance = 0
					Convey("remove TS more than maintenance TS", func() {
						removeMaintenance(&triggerState.MaintenanceInfo, "trigger user", 1500)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						So(actualInfo, ShouldResemble, triggerState.MaintenanceInfo)
						So(actualTS, ShouldResemble, triggerState.Maintenance)
					})
					Convey("remove TS less than maintenance TS", func() {
						removeMaintenance(&triggerState.MaintenanceInfo, "trigger user", 900)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						So(actualTS, ShouldResemble, metricState.Maintenance)
					})
					Convey("remove TS equal to maintenance TS", func() {
						removeMaintenance(&triggerState.MaintenanceInfo, "trigger user", 1000)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						So(actualTS, ShouldResemble, metricState.Maintenance)
					})
				})

				Convey("Metric has removed maintenance trigger has set maintenance", func() {
					metricState.Maintenance = 0
					Convey("remove TS more than maintenance TS", func() {
						removeMaintenance(&metricState.MaintenanceInfo, "metric user", 1500)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						So(actualTS, ShouldResemble, metricState.Maintenance)
					})
					Convey("remove TS less than maintenance TS", func() {
						removeMaintenance(&metricState.MaintenanceInfo, "metric user", 900)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						So(actualInfo, ShouldResemble, triggerState.MaintenanceInfo)
						So(actualTS, ShouldResemble, triggerState.Maintenance)
					})
					Convey("remove TS equal to maintenance TS", func() {
						removeMaintenance(&metricState.MaintenanceInfo, "metric user", 1000)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						So(actualTS, ShouldResemble, metricState.Maintenance)
					})
				})
			})
		})
	})
}

func removeMaintenance(info *moira.MaintenanceInfo, removeUser string, removeTime int64) {
	info.Set(info.StartUser, info.StartTime, &removeUser, &removeTime)
}

func setMaintenance(info *moira.MaintenanceInfo, startUser string, startTime int64) {
	info.Set(&startUser, &startTime, nil, nil)
}
