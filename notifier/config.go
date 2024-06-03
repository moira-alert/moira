package notifier

import (
	"time"
)

const NotificationsLimitUnlimited = int64(-1)

// Config is sending settings including log settings.
type Config struct {
	Enabled                       bool
	SelfStateEnabled              bool
	SelfStateContacts             []map[string]string
	SendingTimeout                time.Duration
	ResendingTimeout              time.Duration
	Senders                       []map[string]interface{}
	LogFile                       string
	LogLevel                      string
	FrontURL                      string
	Location                      *time.Location
	DateTimeFormat                string
	ReadBatchSize                 int64
	MaxFailAttemptToSendAvailable int
	LogContactsToLevel            map[string]string
	LogSubscriptionsToLevel       map[string]string
}
