package api

import (
	"net/http"
	"time"
)

// WebContact is container for web ui contact validation.
type WebContact struct {
	ContactType     string `json:"type" example:"webhook"`
	ContactLabel    string `json:"label" example:"Webhook"`
	ValidationRegex string `json:"validation,omitempty" example:"^(http|https):\\/\\/.*(moira.ru)(:[0-9]{2,5})?\\/"`
	Placeholder     string `json:"placeholder,omitempty" example:"https://moira.ru/webhooks"`
	Help            string `json:"help,omitempty" example:"### Domains whitelist:\n - moira.ru\n"`
}

// FeatureFlags is struct to manage feature flags.
type FeatureFlags struct {
	IsPlottingDefaultOn              bool `json:"isPlottingDefaultOn" example:"false"`
	IsPlottingAvailable              bool `json:"isPlottingAvailable" example:"true"`
	IsSubscriptionToAllTagsAvailable bool `json:"isSubscriptionToAllTagsAvailable" example:"false"`
	IsReadonlyEnabled                bool `json:"isReadonlyEnabled" example:"false"`
}

// Sentry - config for sentry settings
type Sentry struct {
	DSN      string `json:"dsn,omitempty" example:"https://secret@sentry.host"`
	Platform string `json:"platform,omitempty" example:"dev"`
}

// Config for api configuration variables.
type Config struct {
	EnableCORS                bool
	Listen                    string
	GraphiteLocalMetricTTL    time.Duration
	GraphiteRemoteMetricTTL   time.Duration
	PrometheusRemoteMetricTTL time.Duration
	Flags                     FeatureFlags
}

// WebConfig is container for web ui configuration parameters.
type WebConfig struct {
	SupportEmail  string       `json:"supportEmail,omitempty" example:"opensource@skbkontur.com"`
	RemoteAllowed bool         `json:"remoteAllowed" example:"true"`
	Contacts      []WebContact `json:"contacts"`
	FeatureFlags  FeatureFlags `json:"featureFlags"`
	Sentry        Sentry       `json:"sentry"`
}

func (WebConfig) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
