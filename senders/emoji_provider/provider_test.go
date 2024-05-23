package emoji_provider

import (
	"testing"

	"github.com/moira-alert/moira"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewEmojiProvider(t *testing.T) {
	Convey("Successful creation, no errors", t, func() {
		em, err := NewEmojiProvider(
			"default",
			map[string]string{
				"OK":        "super_ok",
				"WARN":      "super_warn",
				"ERROR":     "super_error",
				"TEST":      "super_test",
				"EXCEPTION": "super_exception",
				"NODATA":    "super_nodata",
			},
		)
		So(err, ShouldBeNil)
		expected := &emojiProvider{
			defaultValue: "default",
			stateEmojiMap: map[moira.State]string{
				"OK":        "super_ok",
				"WARN":      "super_warn",
				"ERROR":     "super_error",
				"TEST":      "super_test",
				"EXCEPTION": "super_exception",
				"NODATA":    "super_nodata",
			},
		}
		So(em, ShouldResemble, expected)
	})

	Convey("Unsuccessful creation, has error", t, func() {
		em, err := NewEmojiProvider(
			"default",
			map[string]string{
				"OK":        "super_ok",
				"dfgdf":     "super_warn",
				"ERROR":     "super_error",
				"TEST":      "super_test",
				"EXCEPTION": "super_exception",
				"NODATA":    "super_nodata",
			},
		)
		So(err.Error(), ShouldResemble, "undefined Moira's state: dfgdf")
		So(em, ShouldBeNil)
	})
}

func TestEmojiProvider_GetStateEmoji(t *testing.T) {
	Convey("Check state emoji", t, func() {
		em := &emojiProvider{stateEmojiMap: defaultStateEmoji, defaultValue: "default_value"}
		So(em.GetStateEmoji(moira.StateOK), ShouldResemble, ":moira-state-ok:")
		So(em.GetStateEmoji(moira.StateWARN), ShouldResemble, ":moira-state-warn:")
		So(em.GetStateEmoji(moira.StateERROR), ShouldResemble, ":moira-state-error:")
		So(em.GetStateEmoji(moira.StateNODATA), ShouldResemble, ":moira-state-nodata:")
		So(em.GetStateEmoji(moira.StateEXCEPTION), ShouldResemble, ":moira-state-exception:")
		So(em.GetStateEmoji(moira.StateTEST), ShouldResemble, ":moira-state-test:")
		So(em.GetStateEmoji("dfdfdf"), ShouldResemble, "default_value")
	})
}
