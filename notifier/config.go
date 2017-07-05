package notifier

import "time"

type Config struct {
	LogFile          string
	LogLevel         string
	LogColor         bool
	SendingTimeout   time.Duration
	ResendingTimeout time.Duration
	Senders          []map[string]string
}
