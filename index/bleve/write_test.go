package bleve

import (
	"testing"

	"github.com/moira-alert/moira/index/fixtures"
	"github.com/moira-alert/moira/index/mapping"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTriggerIndex_Write(t *testing.T) {
	var newIndex *TriggerIndex
	var err error
	var count int64

	triggerMapping := mapping.BuildIndexMapping(mapping.Trigger{})

	triggerTestCases := fixtures.IndexedTriggerTestCases

	triggerChecksPointers := triggerTestCases.ToTriggerChecks()

	Convey("First of all, create index", t, func() {
		newIndex, err = CreateTriggerIndex(triggerMapping)
		So(newIndex, ShouldHaveSameTypeAs, &TriggerIndex{})
		So(err, ShouldBeNil)

		count, err = newIndex.GetCount()
		So(count, ShouldBeZeroValue)
		So(err, ShouldBeNil)
	})

	Convey("Test write triggers and get count", t, func() {

		Convey("Test write 0 triggers", func() {
			err = newIndex.Write(triggerChecksPointers[0:0])
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldBeZeroValue)
			So(err, ShouldBeNil)
		})

		Convey("Test write 1 trigger", func() {
			err = newIndex.Write(triggerChecksPointers[0:1])
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(1))
			So(err, ShouldBeNil)
		})

		Convey("Test write the same 1 trigger", func() {
			err = newIndex.Write(triggerChecksPointers[0:1])
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(1))
			So(err, ShouldBeNil)
		})

		Convey("Test write 10 triggers", func() {
			err = newIndex.Write(triggerChecksPointers[0:10])
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(10))
			So(err, ShouldBeNil)
		})

		Convey("Test write the same 10 triggers", func() {
			err = newIndex.Write(triggerChecksPointers[0:10])
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(10))
			So(err, ShouldBeNil)
		})

		Convey("Test write all 31 triggers", func() {
			err = newIndex.Write(triggerChecksPointers)
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(32))
			So(err, ShouldBeNil)
		})

		Convey("Test write the same 31 triggers", func() {
			err = newIndex.Write(triggerChecksPointers)
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(32))
			So(err, ShouldBeNil)
		})

	})
}
