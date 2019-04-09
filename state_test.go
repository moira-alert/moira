package moira

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTtlState_ToMetricState(t *testing.T) {
	Convey("ToMetricState test", t, func(c C) {
		c.So(TTLStateDEL.ToMetricState(), ShouldResemble, StateNODATA)
		c.So(TTLStateOK.ToMetricState(), ShouldResemble, StateOK)
		c.So(TTLStateWARN.ToMetricState(), ShouldResemble, StateWARN)
		c.So(TTLStateERROR.ToMetricState(), ShouldResemble, StateERROR)
		c.So(TTLStateNODATA.ToMetricState(), ShouldResemble, StateNODATA)
	})
}

func TestTtlState_ToTriggerState(t *testing.T) {
	Convey("ToTriggerState test", t, func(c C) {
		c.So(TTLStateDEL.ToTriggerState(), ShouldResemble, StateOK)
		c.So(TTLStateOK.ToTriggerState(), ShouldResemble, StateOK)
		c.So(TTLStateWARN.ToTriggerState(), ShouldResemble, StateWARN)
		c.So(TTLStateERROR.ToTriggerState(), ShouldResemble, StateERROR)
		c.So(TTLStateNODATA.ToTriggerState(), ShouldResemble, StateNODATA)
	})
}
