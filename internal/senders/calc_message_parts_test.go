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

		Convey("descLen > maxChars/2 && eventsLen <= maxChars/2", func() {
			descNewLen, eventsNewLen := CalculateMessagePartsLength(100, 70, 40)
			So(descNewLen, ShouldResemble, 50)
			So(eventsNewLen, ShouldResemble, 40)
		})

		Convey("eventsLen > maxChars/2 && descLen <= maxChars/2", func() {
			descNewLen, eventsNewLen := CalculateMessagePartsLength(100, 40, 70)
			So(descNewLen, ShouldResemble, 40)
			So(eventsNewLen, ShouldResemble, 60)
		})

		Convey("Both greater than maxChars/2", func() {
			descNewLen, eventsNewLen := CalculateMessagePartsLength(100, 70, 70)
			So(descNewLen, ShouldResemble, 40)
			So(eventsNewLen, ShouldResemble, 50)
		})
	})
}
