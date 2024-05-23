package emoji_moderator

import (
	"fmt"
	"maps"

	"github.com/moira-alert/moira"
)

var defaultStateEmoji = map[moira.State]string{
	moira.StateOK:        ":moira-state-ok:",
	moira.StateWARN:      ":moira-state-warn:",
	moira.StateERROR:     ":moira-state-error:",
	moira.StateNODATA:    ":moira-state-nodata:",
	moira.StateEXCEPTION: ":moira-state-exception:",
	moira.StateTEST:      ":moira-state-test:",
}

// emojiModerator is struct for get emoji by trigger State.
type emojiModerator struct {
	defaultValue  string
	stateEmojiMap map[moira.State]string
}

// EmojiModeratorer is interface for emojiModerator.
type EmojiModeratorer interface {
	GetStateEmoji(subjectState moira.State) string
}

// NewEmojiModerator is construct for emojiModerator.
func NewEmojiModerator(defaultValue string, stateEmojiMap map[string]string) (EmojiModeratorer, error) {
	emojiMap := maps.Clone(defaultStateEmoji)

	for state, emoji := range stateEmojiMap {
		converted := moira.State(state)
		if _, ok := emojiMap[converted]; !ok {
			return nil, fmt.Errorf("undefined Moira's state: %s", state)
		}
		emojiMap[converted] = emoji
	}

	return &emojiModerator{
		defaultValue:  defaultValue,
		stateEmojiMap: emojiMap,
	}, nil
}

// GetStateEmoji returns corresponding state emoji.
func (em *emojiModerator) GetStateEmoji(subjectState moira.State) string {
	if emoji, ok := em.stateEmojiMap[subjectState]; ok {
		return emoji
	}

	return em.defaultValue
}
