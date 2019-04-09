package checker

import (
	"testing"

	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetMaintenanceInfo(t *testing.T) {
	Convey("Test getting right maintenance info from trigger or metric", t, func(c C) {
		triggerStateWithRemovedMaintenance := moira.CheckData{Maintenance: 0}
		setMaintenance(&triggerStateWithRemovedMaintenance.MaintenanceInfo, "start trigger user", 100)
		removeMaintenance(&triggerStateWithRemovedMaintenance.MaintenanceInfo, "remove triggre user", 1000)

		metricStateWithRemovedMaintenance := moira.MetricState{Maintenance: 0}
		setMaintenance(&metricStateWithRemovedMaintenance.MaintenanceInfo, "start metric user", 100)
		removeMaintenance(&metricStateWithRemovedMaintenance.MaintenanceInfo, "remove metric user", 1000)

		Convey("Metric state is nil", t, func(c C) {
			actualInfo, actualTS := getMaintenanceInfo(&triggerStateWithRemovedMaintenance, nil)
			c.So(actualInfo, ShouldResemble, triggerStateWithRemovedMaintenance.MaintenanceInfo)
			c.So(actualTS, ShouldResemble, triggerStateWithRemovedMaintenance.Maintenance)
		})

		Convey("Trigger state is nil", t, func(c C) {
			actualInfo, actualTS := getMaintenanceInfo(nil, &metricStateWithRemovedMaintenance)
			c.So(actualInfo, ShouldResemble, metricStateWithRemovedMaintenance.MaintenanceInfo)
			c.So(actualTS, ShouldResemble, metricStateWithRemovedMaintenance.Maintenance)
		})

		Convey("Trigger and Metric state has data", t, func(c C) {
			Convey("Trigger never be in maintenance but metric with maintenance", t, func(c C) {
				triggerState := moira.CheckData{}
				Convey("in maintenance", t, func(c C) {
					metricState := moira.MetricState{Maintenance: 1000}
					setMaintenance(&metricState.MaintenanceInfo, "user", 100)
					actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
					c.So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
					c.So(actualTS, ShouldResemble, metricState.Maintenance)
				})

				Convey("removed maintenance", t, func(c C) {
					actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricStateWithRemovedMaintenance)
					c.So(actualInfo, ShouldResemble, metricStateWithRemovedMaintenance.MaintenanceInfo)
					c.So(actualTS, ShouldResemble, metricStateWithRemovedMaintenance.Maintenance)
				})
			})

			Convey("Metric never be in maintenance but trigger with maintenance", t, func(c C) {
				metricState := moira.MetricState{}
				Convey("in maintenance", t, func(c C) {
					triggerState := moira.CheckData{Maintenance: 1000}
					setMaintenance(&triggerState.MaintenanceInfo, "user", 100)
					actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
					c.So(actualInfo, ShouldResemble, triggerState.MaintenanceInfo)
					c.So(actualTS, ShouldResemble, triggerState.Maintenance)
				})

				Convey("removed maintenance", t, func(c C) {
					actualInfo, actualTS := getMaintenanceInfo(&triggerStateWithRemovedMaintenance, &metricState)
					c.So(actualInfo, ShouldResemble, triggerStateWithRemovedMaintenance.MaintenanceInfo)
					c.So(actualTS, ShouldResemble, triggerStateWithRemovedMaintenance.Maintenance)
				})
			})

			Convey("Trigger and metric has maintenance", t, func(c C) {
				triggerState := moira.CheckData{Maintenance: 1000}
				setMaintenance(&triggerState.MaintenanceInfo, "trigger user", 100)
				metricState := moira.MetricState{Maintenance: 1000}
				setMaintenance(&metricState.MaintenanceInfo, "metric user", 200)
				Convey("Both has set maintenance", t, func(c C) {
					Convey("maintenance TS are equal", t, func(c C) {
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						c.So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						c.So(actualTS, ShouldResemble, metricState.Maintenance)
					})
					Convey("metric maintenance TS are more", t, func(c C) {
						metricState.Maintenance = 2000
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						c.So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						c.So(actualTS, ShouldResemble, metricState.Maintenance)
					})
					Convey("trigger maintenance TS are more", t, func(c C) {
						triggerState.Maintenance = 2000
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						c.So(actualInfo, ShouldResemble, triggerState.MaintenanceInfo)
						c.So(actualTS, ShouldResemble, triggerState.Maintenance)
					})
				})

				Convey("Both has removed maintenance, compare remove time", t, func(c C) {
					metricState.Maintenance = 0
					triggerState.Maintenance = 0
					Convey("remove TS are equal", t, func(c C) {
						removeMaintenance(&metricState.MaintenanceInfo, "metric user", 1500)
						removeMaintenance(&triggerState.MaintenanceInfo, "trigger user", 1500)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						c.So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						c.So(actualTS, ShouldResemble, metricState.Maintenance)
					})
					Convey("metric remove TS are more", t, func(c C) {
						removeMaintenance(&metricState.MaintenanceInfo, "metric user", 1600)
						removeMaintenance(&triggerState.MaintenanceInfo, "trigger user", 1500)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						c.So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						c.So(actualTS, ShouldResemble, metricState.Maintenance)
					})
					Convey("trigger remove TS are more", t, func(c C) {
						removeMaintenance(&metricState.MaintenanceInfo, "metric user", 1400)
						removeMaintenance(&triggerState.MaintenanceInfo, "trigger user", 1500)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						c.So(actualInfo, ShouldResemble, triggerState.MaintenanceInfo)
						c.So(actualTS, ShouldResemble, triggerState.Maintenance)
					})
				})

				Convey("Trigger has removed maintenance metric has set maintenance", t, func(c C) {
					triggerState.Maintenance = 0
					Convey("remove TS more than maintenance TS", t, func(c C) {
						removeMaintenance(&triggerState.MaintenanceInfo, "trigger user", 1500)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						c.So(actualInfo, ShouldResemble, triggerState.MaintenanceInfo)
						c.So(actualTS, ShouldResemble, triggerState.Maintenance)
					})
					Convey("remove TS less than maintenance TS", t, func(c C) {
						removeMaintenance(&triggerState.MaintenanceInfo, "trigger user", 900)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						c.So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						c.So(actualTS, ShouldResemble, metricState.Maintenance)
					})
					Convey("remove TS equal to maintenance TS", t, func(c C) {
						removeMaintenance(&triggerState.MaintenanceInfo, "trigger user", 1000)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						c.So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						c.So(actualTS, ShouldResemble, metricState.Maintenance)
					})
				})

				Convey("Metric has removed maintenance trigger has set maintenance", t, func(c C) {
					metricState.Maintenance = 0
					Convey("remove TS more than maintenance TS", t, func(c C) {
						removeMaintenance(&metricState.MaintenanceInfo, "metric user", 1500)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						c.So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						c.So(actualTS, ShouldResemble, metricState.Maintenance)
					})
					Convey("remove TS less than maintenance TS", t, func(c C) {
						removeMaintenance(&metricState.MaintenanceInfo, "metric user", 900)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						c.So(actualInfo, ShouldResemble, triggerState.MaintenanceInfo)
						c.So(actualTS, ShouldResemble, triggerState.Maintenance)
					})
					Convey("remove TS equal to maintenance TS", t, func(c C) {
						removeMaintenance(&metricState.MaintenanceInfo, "metric user", 1000)
						actualInfo, actualTS := getMaintenanceInfo(&triggerState, &metricState)
						c.So(actualInfo, ShouldResemble, metricState.MaintenanceInfo)
						c.So(actualTS, ShouldResemble, metricState.Maintenance)
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
