package emoji_provider

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

// emojiProvider is struct for get emoji by trigger State.
type emojiProvider struct {
	defaultValue  string
	stateEmojiMap map[moira.State]string
}

// StateEmojiGetter is interface for emojiProvider.
type StateEmojiGetter interface {
	GetStateEmoji(subjectState moira.State) string
}

// NewEmojiProvider is construct for emojiProvider.
func NewEmojiProvider(defaultValue string, stateEmojiMap map[string]string) (StateEmojiGetter, error) {
	emojiMap := maps.Clone(defaultStateEmoji)

	for state, emoji := range stateEmojiMap {
		converted := moira.State(state)
		if _, ok := emojiMap[converted]; !ok {
			return nil, fmt.Errorf("undefined Moira's state: %s", state)
		}
		emojiMap[converted] = emoji
	}

	return &emojiProvider{
		defaultValue:  defaultValue,
		stateEmojiMap: emojiMap,
	}, nil
}

// GetStateEmoji returns corresponding state emoji.
func (em *emojiProvider) GetStateEmoji(subjectState moira.State) string {
	if emoji, ok := em.stateEmojiMap[subjectState]; ok {
		return emoji
	}

	return em.defaultValue
}
