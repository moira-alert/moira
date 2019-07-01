package notifier

import (
	"time"
)

// Config is sending settings including log settings
type Config struct {
	Enabled          bool
	SendingTimeout   time.Duration
	ResendingTimeout time.Duration
	Senders          []map[string]string
	ImageStore       map[string]string
	LogFile          string
	LogLevel         string
	FrontURL         string
	Location         *time.Location
	DateTimeFormat   string
}
