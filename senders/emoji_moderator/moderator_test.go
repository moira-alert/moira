package emoji_moderator

import (
	"testing"

	"github.com/moira-alert/moira"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetStateEmoji(t *testing.T) {
	Convey("Check state emoji", t, func() {
		So(GetStateEmoji(moira.StateOK, ""), ShouldResemble, okEmoji)
		So(GetStateEmoji(moira.StateWARN, ""), ShouldResemble, warnEmoji)
		So(GetStateEmoji(moira.StateERROR, ""), ShouldResemble, errorEmoji)
		So(GetStateEmoji(moira.StateNODATA, ""), ShouldResemble, nodataEmoji)
		So(GetStateEmoji(moira.StateEXCEPTION, ""), ShouldResemble, exceptionEmoji)
		So(GetStateEmoji(moira.StateTEST, ""), ShouldResemble, testEmoji)
		So(GetStateEmoji("dfdfdf", "default_value"), ShouldResemble, "default_value")
	})
}
