package emoji_moderator

import (
	"github.com/moira-alert/moira"
)

const (
	okEmoji        = ":moira-state-ok:"
	warnEmoji      = ":moira-state-warn:"
	errorEmoji     = ":moira-state-error:"
	nodataEmoji    = ":moira-state-nodata:"
	exceptionEmoji = ":moira-state-exception:"
	testEmoji      = ":moira-state-test:"
)

var stateEmoji = map[moira.State]string{
	moira.StateOK:        okEmoji,
	moira.StateWARN:      warnEmoji,
	moira.StateERROR:     errorEmoji,
	moira.StateNODATA:    nodataEmoji,
	moira.StateEXCEPTION: exceptionEmoji,
	moira.StateTEST:      testEmoji,
}

// GetStateEmoji returns corresponding state emoji.
func GetStateEmoji(subjectState moira.State, defaultValue string) string {
	if emoji, ok := stateEmoji[subjectState]; ok {
		return emoji
	}

	return defaultValue
}
