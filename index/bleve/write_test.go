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

	Convey("First of all, create index", t, func(c C) {
		newIndex, err = CreateTriggerIndex(triggerMapping)
		c.So(newIndex, ShouldHaveSameTypeAs, &TriggerIndex{})
		c.So(err, ShouldBeNil)

		count, err = newIndex.GetCount()
		c.So(count, ShouldBeZeroValue)
		c.So(err, ShouldBeNil)
	})

	Convey("Test write triggers and get count", t, func(c C) {

		Convey("Test write 0 triggers", t, func(c C) {
			err = newIndex.Write(triggerChecksPointers[0:0])
			c.So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			c.So(count, ShouldBeZeroValue)
			c.So(err, ShouldBeNil)
		})

		Convey("Test write 1 trigger", t, func(c C) {
			err = newIndex.Write(triggerChecksPointers[0:1])
			c.So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			c.So(count, ShouldEqual, int64(1))
			c.So(err, ShouldBeNil)
		})

		Convey("Test write the same 1 trigger", t, func(c C) {
			err = newIndex.Write(triggerChecksPointers[0:1])
			c.So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			c.So(count, ShouldEqual, int64(1))
			c.So(err, ShouldBeNil)
		})

		Convey("Test write 10 triggers", t, func(c C) {
			err = newIndex.Write(triggerChecksPointers[0:10])
			c.So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			c.So(count, ShouldEqual, int64(10))
			c.So(err, ShouldBeNil)
		})

		Convey("Test write the same 10 triggers", t, func(c C) {
			err = newIndex.Write(triggerChecksPointers[0:10])
			c.So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			c.So(count, ShouldEqual, int64(10))
			c.So(err, ShouldBeNil)
		})

		Convey("Test write all 31 triggers", t, func(c C) {
			err = newIndex.Write(triggerChecksPointers)
			c.So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			c.So(count, ShouldEqual, int64(32))
			c.So(err, ShouldBeNil)
		})

		Convey("Test write the same 31 triggers", t, func(c C) {
			err = newIndex.Write(triggerChecksPointers)
			c.So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			c.So(count, ShouldEqual, int64(32))
			c.So(err, ShouldBeNil)
		})

	})
}
