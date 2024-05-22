package emoji_moderator

import (
	"fmt"
	"github.com/moira-alert/moira"
	"maps"
)

var defaultStateEmoji = map[moira.State]string{
	moira.StateOK:        ":moira-state-ok:",
	moira.StateWARN:      ":moira-state-warn:",
	moira.StateERROR:     ":moira-state-error:",
	moira.StateNODATA:    ":moira-state-nodata:",
	moira.StateEXCEPTION: ":moira-state-exception:",
	moira.StateTEST:      ":moira-state-test:",
}

// EmojiModerator is struct for get emoji by trigger State.
type EmojiModerator struct {
	defaultValue  string
	stateEmojiMap map[moira.State]string
}

// NewEmojiModerator is construct for EmojiModerator.
func NewEmojiModerator(defaultValue string, stateEmojiMap map[string]string) (*EmojiModerator, error) {
	emojiMap := maps.Clone(defaultStateEmoji)

	for state, emoji := range stateEmojiMap {
		converted := moira.State(state)
		if _, ok := emojiMap[converted]; !ok {
			return nil, fmt.Errorf("undefined Moira's state: %s", state)
		}
		emojiMap[converted] = emoji
	}

	return &EmojiModerator{
		defaultValue:  defaultValue,
		stateEmojiMap: emojiMap,
	}, nil
}

// GetStateEmoji returns corresponding state emoji.
func (em *EmojiModerator) GetStateEmoji(subjectState moira.State, defaultValue string) string {
	if emoji, ok := em.stateEmojiMap[subjectState]; ok {
		return emoji
	}

	return defaultValue
}
