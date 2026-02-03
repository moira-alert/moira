package redis

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/moira-alert/moira/clock"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDeliveryChecksDataManipulation(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger, clock.NewSystemClock())
	dataBase.Flush()

	defer dataBase.Flush()

	const testContactType = "test_contact_type"

	var testTimestamp int64 = 12345678

	storedData := make([]string, 0)

	Convey("Test delivery checks manipulation", t, func() {
		Convey("store delivery checks data", func() {
			for i := range 6 {
				data := fmt.Sprintf("data_%v", i)
				storedData = append(storedData, data)

				err := dataBase.AddDeliveryChecksData(testContactType, testTimestamp+int64(i), data)
				So(err, ShouldBeNil)
			}
		})

		Convey("get delivery checks", func() {
			Convey("all checks", func() {
				gotRes, err := dataBase.GetDeliveryChecksData(testContactType, "-inf", "+inf")
				So(err, ShouldBeNil)
				So(gotRes, ShouldResemble, storedData)
			})

			Convey("with fixed from", func() {
				gotRes, err := dataBase.GetDeliveryChecksData(testContactType, strconv.FormatInt(testTimestamp+2, 10), "+inf")
				So(err, ShouldBeNil)
				So(gotRes, ShouldResemble, storedData[2:])
			})

			Convey("with fixed to", func() {
				gotRes, err := dataBase.GetDeliveryChecksData(testContactType, "-inf", strconv.FormatInt(testTimestamp+4, 10))
				So(err, ShouldBeNil)
				So(gotRes, ShouldResemble, storedData[:5])
			})

			Convey("with fixed from and to", func() {
				gotRes, err := dataBase.GetDeliveryChecksData(testContactType, strconv.FormatInt(testTimestamp+2, 10), strconv.FormatInt(testTimestamp+4, 10))
				So(err, ShouldBeNil)
				So(gotRes, ShouldResemble, storedData[2:5])
			})
		})

		Convey("remove delivery checks", func() {
			removedCount, err := dataBase.RemoveDeliveryChecksData(testContactType, "-inf", strconv.FormatInt(testTimestamp+3, 10))
			So(err, ShouldBeNil)
			So(removedCount, ShouldEqual, 4)

			gotRes, err := dataBase.GetDeliveryChecksData(testContactType, strconv.FormatInt(testTimestamp+2, 10), strconv.FormatInt(testTimestamp+6, 10))
			So(err, ShouldBeNil)
			So(gotRes, ShouldResemble, storedData[4:])
		})
	})
}
