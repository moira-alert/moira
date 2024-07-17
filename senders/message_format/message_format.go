package message_format

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/emoji_provider"
	"time"
)

// MessageFormatter is used for formatting messages to send via telegram, mattermost, etc.
type MessageFormatter interface {
	Format(params MessageFormatterParams) string
}

// MessageFormatterParams is the parameters for MessageFormatter.
type MessageFormatterParams struct {
	Events  moira.NotificationEvents
	Trigger moira.TriggerData
	// EmojiGetter used in titles for better description.
	EmojiGetter emoji_provider.StateEmojiGetter
	FrontURI    string
	// MessageMaxChars is a limit for future message. If -1 then no limit is set.
	MessageMaxChars int64
	Location        *time.Location
	Throttled       bool
}
