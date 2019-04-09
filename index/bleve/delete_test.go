package bleve

import (
	"testing"

	"github.com/moira-alert/moira/index/fixtures"
	"github.com/moira-alert/moira/index/mapping"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTriggerIndex_Delete(t *testing.T) {
	var newIndex *TriggerIndex
	var err error
	var count int64

	triggerMapping := mapping.BuildIndexMapping(mapping.Trigger{})

	triggerTestCases := fixtures.IndexedTriggerTestCases

	triggerIDs := triggerTestCases.ToTriggerIDs()
	triggerChecksPointers := triggerTestCases.ToTriggerChecks()

	Convey("First of all, create and fill index", t, func(c C) {
		newIndex, err = CreateTriggerIndex(triggerMapping)
		c.So(newIndex, ShouldHaveSameTypeAs, &TriggerIndex{})
		c.So(err, ShouldBeNil)

		count, err = newIndex.GetCount()
		c.So(count, ShouldBeZeroValue)
		c.So(err, ShouldBeNil)

		err = newIndex.Write(triggerChecksPointers)
		c.So(err, ShouldBeNil)

		count, err = newIndex.GetCount()
		c.So(count, ShouldEqual, int64(32))
		c.So(err, ShouldBeNil)
	})

	Convey("Test remove trigger IDs from index", t, func(c C) {
		Convey("Remove 0 trigger IDs", t, func(c C) {
			err = newIndex.Delete(triggerIDs[0:0])
			c.So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			c.So(count, ShouldEqual, int64(32))
			c.So(err, ShouldBeNil)
		})

		Convey("Remove 1 trigger ID", t, func(c C) {
			err = newIndex.Delete(triggerIDs[0:1])
			c.So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			c.So(count, ShouldEqual, int64(31))
			c.So(err, ShouldBeNil)
		})

		Convey("Remove the same 1 trigger ID", t, func(c C) {
			err = newIndex.Delete(triggerIDs[0:1])
			c.So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			c.So(count, ShouldEqual, int64(31))
			c.So(err, ShouldBeNil)
		})

		Convey("Remove 10 trigger IDs", t, func(c C) {
			err = newIndex.Delete(triggerIDs[0:10])
			c.So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			c.So(count, ShouldEqual, int64(22))
			c.So(err, ShouldBeNil)
		})

		Convey("Remove the same 10 trigger IDs", t, func(c C) {
			err = newIndex.Delete(triggerIDs[0:10])
			c.So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			c.So(count, ShouldEqual, int64(22))
			c.So(err, ShouldBeNil)
		})

		Convey("Remove all 32 trigger IDs", t, func(c C) {
			err = newIndex.Delete(triggerIDs)
			c.So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			c.So(count, ShouldEqual, int64(0))
			c.So(err, ShouldBeNil)
		})
	})
}
