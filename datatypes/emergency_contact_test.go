package datatypes

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestIsValidHeartbeatType(t *testing.T) {
	Convey("Test IsValid heartbeat type", t, func() {
		Convey("Test valid cases", func() {
			testcases := []HeartbeatType{
				HeartbeatNotifierOff,
			}

			for _, testcase := range testcases {
				So(testcase.IsValid(), ShouldBeTrue)
			}
		})

		Convey("Test invalid cases", func() {
			testcases := []HeartbeatType{
				"notifier_on",
				"checker_off",
			}

			for _, testcase := range testcases {
				So(testcase.IsValid(), ShouldBeFalse)
			}
		})
	})
}
