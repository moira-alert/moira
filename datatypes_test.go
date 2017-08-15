package moira

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestIsScheduleAllows(t *testing.T) {
	allDaysExcludedSchedule := ScheduleData{
		TimezoneOffset: -300,
		StartOffset:    0,
		EndOffset:      1439,
		Days: []ScheduleDataDay{
			{
				Name:    "Mon",
				Enabled: false,
			},
			{
				Name:    "Tue",
				Enabled: false,
			},
			{
				Name:    "Wed",
				Enabled: false,
			},
			{
				Name:    "Thu",
				Enabled: false,
			},
			{
				Name:    "Fri",
				Enabled: false,
			},
			{
				Name:    "Sat",
				Enabled: false,
			},
			{
				Name:    "Sun",
				Enabled: false,
			},
		},
	}

	//367980 - 01/05/1970 6:13am (UTC) Mon
	//454380 - 01/06/1970 6:13am (UTC) Tue

	Convey("No schedule", t, func() {
		var noSchedule *ScheduleData = nil
		So(noSchedule.IsScheduleAllows(367980), ShouldBeTrue)
	})

	Convey("Full schedule", t, func() {
		schedule := getDefaultSchedule()
		So(schedule.IsScheduleAllows(367980), ShouldBeTrue)
	})

	Convey("Exclude monday", t, func() {
		schedule := getDefaultSchedule()
		schedule.Days[0].Enabled = false
		So(schedule.IsScheduleAllows(367980), ShouldBeFalse)
		So(schedule.IsScheduleAllows(367980 + 86400), ShouldBeTrue)
		So(schedule.IsScheduleAllows(367980 + 86400 * 2), ShouldBeTrue)
	})

	Convey("Exclude all days", t, func() {
		schedule := allDaysExcludedSchedule
		So(schedule.IsScheduleAllows(367980), ShouldBeFalse)
		So(schedule.IsScheduleAllows(367980 + 86400), ShouldBeFalse)
		So(schedule.IsScheduleAllows(367980 + 86400 * 5), ShouldBeFalse)
	})

	Convey("Include only morning", t, func() {
		schedule := getDefaultSchedule()
		schedule.StartOffset = 60
		schedule.EndOffset = 540
		So(schedule.IsScheduleAllows(86400+129*60), ShouldBeTrue)  // 2/01/1970 2:09
		So(schedule.IsScheduleAllows(86400-239*60), ShouldBeTrue)  // 1/01/1970 20:01
		So(schedule.IsScheduleAllows(86400-241*60), ShouldBeFalse) // 1/01/1970 19:58
		So(schedule.IsScheduleAllows(86400+541*60), ShouldBeFalse) // 2/01/1970 9:01
		So(schedule.IsScheduleAllows(86400-255*60), ShouldBeFalse) // 1/01/1970 19:45
	})

	Convey("Exclude morning", t, func() {
		schedule := getDefaultSchedule()
		schedule.StartOffset = 540
		schedule.EndOffset = 1499
		So(schedule.IsScheduleAllows(86400+129*60), ShouldBeFalse) // 2/01/1970 2:09
		So(schedule.IsScheduleAllows(86400-239*60), ShouldBeFalse) // 1/01/1970 20:01
		So(schedule.IsScheduleAllows(86400-242*60), ShouldBeTrue)  // 1/01/1970 19:58
		So(schedule.IsScheduleAllows(86400+541*60), ShouldBeTrue)  // 2/01/1970 9:01
		So(schedule.IsScheduleAllows(86400-255*60), ShouldBeTrue)  // 1/01/1970 19:45
	})
}

func getDefaultSchedule() ScheduleData {
	return ScheduleData{
		TimezoneOffset: -300,
		StartOffset:    0,
		EndOffset:      1439,
		Days: []ScheduleDataDay{
			{
				Name:    "Mon",
				Enabled: true,
			},
			{
				Name:    "Tue",
				Enabled: true,
			},
			{
				Name:    "Wed",
				Enabled: true,
			},
			{
				Name:    "Thu",
				Enabled: true,
			},
			{
				Name:    "Fri",
				Enabled: true,
			},
			{
				Name:    "Sat",
				Enabled: true,
			},
			{
				Name:    "Sun",
				Enabled: true,
			},
		},
	}
}
