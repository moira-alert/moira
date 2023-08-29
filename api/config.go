package api

import "time"

// Config for api configuration variables.
type Config struct {
	EnableCORS                bool
	Listen                    string
	GraphiteLocalMetricTTL    time.Duration
	GraphiteRemoteMetricTTL   time.Duration
	PrometheusRemoteMetricTTL time.Duration
}

// WebConfig is container for web ui configuration parameters.
type WebConfig struct {
	SupportEmail  string       `json:"supportEmail,omitempty"`
	RemoteAllowed bool         `json:"remoteAllowed"`
	Contacts      []WebContact `json:"contacts"`
	FeatureFlags  FeatureFlags `json:"featureFlags"`
}

// WebContact is container for web ui contact validation.
type WebContact struct {
	ContactType     string `json:"type"`
	ContactLabel    string `json:"label"`
	ValidationRegex string `json:"validation,omitempty"`
	Placeholder     string `json:"placeholder,omitempty"`
	Help            string `json:"help,omitempty"`
}

// FeatureFlags is struct to manage feature flags.
type FeatureFlags struct {
	IsPlottingDefaultOn              bool `json:"isPlottingDefaultOn"`
	IsPlottingAvailable              bool `json:"isPlottingAvailable"`
	IsSubscriptionToAllTagsAvailable bool `json:"isSubscriptionToAllTagsAvailable"`
}
