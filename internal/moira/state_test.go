package moira

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTtlState_ToMetricState(t *testing.T) {
	Convey("ToMetricState test", t, func() {
		So(TTLStateDEL.ToMetricState(), ShouldResemble, StateNODATA)
		So(TTLStateOK.ToMetricState(), ShouldResemble, StateOK)
		So(TTLStateWARN.ToMetricState(), ShouldResemble, StateWARN)
		So(TTLStateERROR.ToMetricState(), ShouldResemble, StateERROR)
		So(TTLStateNODATA.ToMetricState(), ShouldResemble, StateNODATA)
	})
}

func TestTtlState_ToTriggerState(t *testing.T) {
	Convey("ToTriggerState test", t, func() {
		So(TTLStateDEL.ToTriggerState(), ShouldResemble, StateOK)
		So(TTLStateOK.ToTriggerState(), ShouldResemble, StateOK)
		So(TTLStateWARN.ToTriggerState(), ShouldResemble, StateWARN)
		So(TTLStateERROR.ToTriggerState(), ShouldResemble, StateERROR)
		So(TTLStateNODATA.ToTriggerState(), ShouldResemble, StateNODATA)
	})
}
