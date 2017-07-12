package notifier

import "time"

//Config is sending settings including log settings
type Config struct {
	LogFile          string
	LogLevel         string
	LogColor         bool
	SendingTimeout   time.Duration
	ResendingTimeout time.Duration
	Senders          []map[string]string
}
