package api

import "time"

// Config for api configuration variables
type Config struct {
	EnableCORS      bool
	Listen          string
	LocalMetricTTL  time.Duration
	RemoteMetricTTL time.Duration
}

// WebConfig is container for web ui configuration parameters
type WebConfig struct {
	SupportEmail  string       `json:"supportEmail,omitempty"`
	RemoteAllowed bool         `json:"remoteAllowed"`
	Contacts      []WebContact `json:"contacts"`
}

// WebContact is container for web ui contact validation
type WebContact struct {
	ContactType     string `json:"type"`
	ContactLabel    string `json:"label"`
	ValidationRegex string `json:"validation,omitempty"`
	Placeholder     string `json:"placeholder,omitempty"`
	Help            string `json:"help,omitempty"`
}
