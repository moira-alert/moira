package telegram

import "github.com/moira-alert/moira"

var (
	emojiStates = map[moira.State]string{
		moira.StateOK:     "\xe2\x9c\x85",
		moira.StateWARN:   "\xe2\x9a\xa0",
		moira.StateERROR:  "\xe2\xad\x95",
		moira.StateNODATA: "\xf0\x9f\x92\xa3",
		moira.StateTEST:   "\xf0\x9f\x98\x8a",
	}
)

type telegramEmojiProvider struct{}

func (_ telegramEmojiProvider) GetStateEmoji(subjectState moira.State) string {
	return emojiStates[subjectState]
}
