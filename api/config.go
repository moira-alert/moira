package api

import (
	"net/http"
	"time"

	"github.com/moira-alert/moira"
)

// WebContact is container for web ui contact validation.
type WebContact struct {
	ContactType     string `json:"type" example:"webhook"`
	ContactLabel    string `json:"label" example:"Webhook"`
	LogoURI         string `json:"logo_uri,omitempty" example:"discord-logo.svg"`
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

// Sentry - config for sentry settings.
type Sentry struct {
	DSN      string `json:"dsn,omitempty" example:"https://secret@sentry.host"`
	Platform string `json:"platform,omitempty" example:"dev"`
}

// Config for api configuration variables.
type Config struct {
	EnableCORS    bool
	Listen        string
	MetricsTTL    map[moira.ClusterKey]time.Duration
	Flags         FeatureFlags
	Authorization Authorization
}

// Authorization contains authorization configuration.
type Authorization struct {
	AdminList           map[string]struct{}
	Enabled             bool
	AllowedContactTypes map[string]struct{}
}

// IsEnabled returns true if auth is enabled and false otherwise.
func (auth *Authorization) IsEnabled() bool {
	return auth.Enabled
}

// IsAdmin checks whether given user is considered an administrator.
func (auth *Authorization) IsAdmin(login string) bool {
	if !auth.IsEnabled() {
		return false
	}
	_, ok := auth.AdminList[login]
	return ok
}

// WebConfig is container for web ui configuration parameters.
type WebConfig struct {
	SupportEmail         string                `json:"supportEmail,omitempty" example:"opensource@skbkontur.com"`
	RemoteAllowed        bool                  `json:"remoteAllowed" example:"true"`
	MetricSourceClusters []MetricSourceCluster `json:"metric_source_clusters"`
	Contacts             []WebContact          `json:"contacts"`
	FeatureFlags         FeatureFlags          `json:"featureFlags"`
	Sentry               Sentry                `json:"sentry"`
}

// MetricSourceCluster contains data about supported metric source cluster.
type MetricSourceCluster struct {
	TriggerSource moira.TriggerSource `json:"trigger_source" example:"graphite_remote"`
	ClusterId     moira.ClusterId     `json:"cluster_id" example:"default"`
	ClusterName   string              `json:"cluster_name" example:"Graphite Remote Prod"`
}

func (WebConfig) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
