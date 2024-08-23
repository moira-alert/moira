// Package msgformat provides MessageFormatter interface which may be used for formatting messages.
// Also, it contains some realizations such as highlightSyntaxFormatter.
package msgformat

import (
	"github.com/moira-alert/moira"
)

const ChangeTriggerRecommendation = "fix your system or tune this trigger"

// MessageFormatter is used for formatting messages to send via telegram, mattermost, etc.
type MessageFormatter interface {
	Format(params MessageFormatterParams) string
}

// MessageFormatterParams is the parameters for MessageFormatter.
type MessageFormatterParams struct {
	Events  moira.NotificationEvents
	Trigger moira.TriggerData
	// MessageMaxChars is a limit for future message. If -1 then no limit is set.
	MessageMaxChars int
	Throttled       bool
}
