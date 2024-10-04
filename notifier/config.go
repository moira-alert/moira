package notifier

import (
	"time"
)

// There is a duplicate of this constant in database package to prevent cyclic dependencies.
const NotificationsLimitUnlimited = int64(-1)

// Config is sending settings including log settings.
type Config struct {
	Enabled                       bool
	SelfStateEnabled              bool
	SelfStateContacts             []map[string]string
	SendingTimeout                time.Duration
	ResendingTimeout              time.Duration
	ReschedulingDelay             time.Duration
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
