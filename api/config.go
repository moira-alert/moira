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
	IsPlottingDefaultOn              bool            `json:"isPlottingDefaultOn" example:"false"`
	IsPlottingAvailable              bool            `json:"isPlottingAvailable" example:"true"`
	IsSubscriptionToAllTagsAvailable bool            `json:"isSubscriptionToAllTagsAvailable" example:"false"`
	IsReadonlyEnabled                bool            `json:"isReadonlyEnabled" example:"false"`
	CelebrationMode                  CelebrationMode `json:"celebrationMode" swaggertype:"string" example:"new_year"`
}

// CelebrationMode is type for celebrate Moira.
type CelebrationMode string

const newYear CelebrationMode = "new_year"

// AvailableCelebrationMode map with available celebration mode.
var availableCelebrationMode = map[CelebrationMode]struct{}{
	newYear: {},
}

// IsAvailableCelebrationMode return is mode available or not.
func IsAvailableCelebrationMode(mode CelebrationMode) bool {
	_, ok := availableCelebrationMode[mode]

	return ok
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

func (WebConfig) Render(http.ResponseWriter, *http.Request) error {
	return nil
}

const (
	// DefaultTriggerNameMaxSize which will be used while validating dto.Trigger.
	DefaultTriggerNameMaxSize = 200
	// DefaultTriggerPagerTTL which will be used via trigger creation.
	DefaultTriggerPagerTTL = time.Minute * 30
)

// PagerLimits contains all limits applied for pagers.
type PagerLimits struct {
	// TTL is the amount of time that the pager will be exist
	TTL time.Duration
}

// LimitsConfig contains limits for some entities.
type LimitsConfig struct {
	// Pager contains limits for pagers.
	Pager PagerLimits
	// Trigger contains limits for triggers.
	Trigger TriggerLimits
	// Trigger contains limits for teams.
	Team TeamLimits
}

// TriggerLimits contains all limits applied for triggers.
type TriggerLimits struct {
	// MaxNameSize is the amount of characters allowed in trigger name.
	MaxNameSize int
}

// GetTestLimitsConfig is used for testing.
func GetTestLimitsConfig() LimitsConfig {
	return LimitsConfig{
		Pager: PagerLimits{
			TTL: DefaultTriggerPagerTTL,
		},
		Trigger: TriggerLimits{
			MaxNameSize: DefaultTriggerNameMaxSize,
		},
		Team: TeamLimits{
			MaxNameSize:        DefaultTeamNameMaxSize,
			MaxDescriptionSize: DefaultTeamDescriptionMaxSize,
		},
	}
}

const (
	DefaultTeamNameMaxSize        = 100
	DefaultTeamDescriptionMaxSize = 1000
)

// TeamLimits contains all limits applied for triggers.
type TeamLimits struct {
	// MaxNameSize is the amount of characters allowed in team name.
	MaxNameSize int
	// MaxNameSize is the amount of characters allowed in team description.
	MaxDescriptionSize int
}
