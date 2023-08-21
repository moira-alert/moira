package bleve

import (
	"testing"

	"github.com/moira-alert/moira/index/mapping"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTriggerIndex_CreateAndGetCount(t *testing.T) {
	var newIndex *TriggerIndex

	var err error

	triggerMapping := mapping.BuildIndexMapping(mapping.Trigger{})

	Convey("Test create index", t, func() {
		newIndex, err = CreateTriggerIndex(triggerMapping)

		So(newIndex, ShouldHaveSameTypeAs, &TriggerIndex{})
		So(err, ShouldBeNil)

		count, err := newIndex.GetCount()
		So(count, ShouldBeZeroValue)
		So(err, ShouldBeNil)
	})

	Convey("Test close index", t, func() {
		err = newIndex.Close()
		So(err, ShouldBeNil)
	})
}
