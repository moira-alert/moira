package bleve

import (
	"testing"

	"github.com/moira-alert/moira/fixtures"
	"github.com/moira-alert/moira/index/mapping"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTriggerIndex_Delete(t *testing.T) {
	var newIndex *TriggerIndex
	var err error
	var count int64

	triggerMapping := mapping.BuildIndexMapping(mapping.Trigger{})

	triggerTestCases := fixtures.TriggerTestCases

	triggerIDs := triggerTestCases.ToTriggerIDs()
	triggerChecksPointers := triggerTestCases.ToTriggerChecks()

	Convey("First of all, create and fill index", t, func() {
		newIndex, err = CreateTriggerIndex(triggerMapping)
		So(newIndex, ShouldHaveSameTypeAs, &TriggerIndex{})
		So(err, ShouldBeNil)

		count, err = newIndex.GetCount()
		So(count, ShouldBeZeroValue)
		So(err, ShouldBeNil)

		err = newIndex.Write(triggerChecksPointers)
		So(err, ShouldBeNil)

		count, err = newIndex.GetCount()
		So(count, ShouldEqual, int64(31))
		So(err, ShouldBeNil)
	})

	Convey("Test remove trigger IDs from index", t, func() {
		Convey("Remove 0 trigger IDs", func() {
			err = newIndex.Delete(triggerIDs[0:0])
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(31))
			So(err, ShouldBeNil)
		})

		Convey("Remove 1 trigger ID", func() {
			err = newIndex.Delete(triggerIDs[0:1])
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(30))
			So(err, ShouldBeNil)
		})

		Convey("Remove the same 1 trigger ID", func() {
			err = newIndex.Delete(triggerIDs[0:1])
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(30))
			So(err, ShouldBeNil)
		})

		Convey("Remove 10 trigger IDs", func() {
			err = newIndex.Delete(triggerIDs[0:10])
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(21))
			So(err, ShouldBeNil)
		})

		Convey("Remove the same 10 trigger IDs", func() {
			err = newIndex.Delete(triggerIDs[0:10])
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(21))
			So(err, ShouldBeNil)
		})

		Convey("Remove all 31 trigger IDs", func() {
			err = newIndex.Delete(triggerIDs)
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(0))
			So(err, ShouldBeNil)
		})
	})
}
