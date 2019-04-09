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

	Convey("Test create index", t, func(c C) {
		newIndex, err = CreateTriggerIndex(triggerMapping)
		c.So(newIndex, ShouldHaveSameTypeAs, &TriggerIndex{})
		c.So(err, ShouldBeNil)

		count, err := newIndex.GetCount()
		c.So(count, ShouldBeZeroValue)
		c.So(err, ShouldBeNil)
	})
}
