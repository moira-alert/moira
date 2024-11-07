package api

import (
	"net/http"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/datatypes"
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
	Limits        LimitsConfig
}

// WebConfig is container for web ui configuration parameters.
type WebConfig struct {
	SupportEmail          string                    `json:"supportEmail,omitempty" example:"opensource@skbkontur.com"`
	RemoteAllowed         bool                      `json:"remoteAllowed" example:"true"`
	MetricSourceClusters  []MetricSourceCluster     `json:"metric_source_clusters"`
	Contacts              []WebContact              `json:"contacts"`
	EmergencyContactTypes []datatypes.HeartbeatType `json:"emergency_contact_types"`
	FeatureFlags          FeatureFlags              `json:"featureFlags"`
	Sentry                Sentry                    `json:"sentry"`
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

const (
	// DefaultTriggerNameMaxSize which will be used while validating dto.Trigger.
	DefaultTriggerNameMaxSize = 200
)

// LimitsConfig contains limits for some entities.
type LimitsConfig struct {
	// Trigger contains limits for triggers.
	Trigger TriggerLimits
}

// TriggerLimits contains all limits applied for triggers.
type TriggerLimits struct {
	// MaxNameSize is the amount of characters allowed in trigger name.
	MaxNameSize int
}

// GetTestLimitsConfig is used for testing.
func GetTestLimitsConfig() LimitsConfig {
	return LimitsConfig{
		Trigger: TriggerLimits{
			MaxNameSize: DefaultTriggerNameMaxSize,
		},
	}
}
