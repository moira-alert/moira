package senders

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCalculateMessagePartsLength(t *testing.T) {
	Convey("Message parts length calculation tests", t, func() {
		Convey("descLen+eventsLen <= maxChars", func() {
			descNewLen, eventsNewLen := CalculateMessagePartsLength(100, 20, 78)
			So(descNewLen, ShouldResemble, 20)
			So(eventsNewLen, ShouldResemble, 78)
		})

		Convey("eventsLen less than percent for event", func() {
			descNewLen, eventsNewLen := CalculateMessagePartsLength(100, 70, 10)
			So(descNewLen, ShouldResemble, 70)
			So(eventsNewLen, ShouldResemble, 10)
		})

		Convey("eventsLen more than percent for event", func() {
			descNewLen, eventsNewLen := CalculateMessagePartsLength(100, 70, 40)
			So(descNewLen, ShouldResemble, 70)
			So(eventsNewLen, ShouldResemble, 30)
		})

		Convey("Both greater than percent", func() {
			descNewLen, eventsNewLen := CalculateMessagePartsLength(100, 70, 70)
			So(descNewLen, ShouldResemble, 70)
			So(eventsNewLen, ShouldResemble, 30)
		})
	})
}
