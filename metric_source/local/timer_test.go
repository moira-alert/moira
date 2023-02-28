package local

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTimerNumberOfTimeSlots(t *testing.T) {
	retention := int64(60)
	steps := int64(10)

	// Specific edge-case that is required by carbonapi
	Convey("Given `from` is divisible by retention", t, func() {
		for _, from := range []int64{0, retention} {
			until := from + retention*steps
			timer := NewTimerRoundingTimestamps(from, until, retention)

			So(timer.NumberOfTimeSlots(), ShouldEqual, steps+1)
		}
	})

	Convey("Given `from` is divisible by retention", t, func() {
		from := int64(0)
		until := int64(0)
		timer := NewTimerRoundingTimestamps(from, until, retention)

		So(timer.NumberOfTimeSlots(), ShouldEqual, 1)
	})

	Convey("Given `from` is not divisible by retention", t, func() {
		for from := int64(1); from < retention; from++ {
			until := from + retention*steps
			timer := NewTimerRoundingTimestamps(from, until, retention)

			So(timer.NumberOfTimeSlots(), ShouldEqual, steps)
		}
	})
}

func TestTimerGetTimeSlot(t *testing.T) {
	Convey("Given a set of test cases", t, func() {
		retention := int64(10)
		from := int64(10)
		until := int64(60)
		timer := NewTimerRoundingTimestamps(from, until, retention)

		testCases := []struct {
			timestamp int64
			timeSlot  int
		}{
			{10, 0},
			{15, 0},
			{19, 0},
			{20, 1},
			{21, 1},
			{25, 1},
			{29, 1},
			{30, 2},
		}

		for _, testCase := range testCases {
			actual := timer.GetTimeSlot(testCase.timestamp)
			So(actual, ShouldEqual, testCase.timeSlot)
		}
	})
}
