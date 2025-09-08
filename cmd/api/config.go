package main

import (
	"fmt"
	"regexp"
	"time"

	"github.com/moira-alert/moira"
	"github.com/xiam/to"

	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/notifier/selfstate"
)

type config struct {
	Redis               cmd.RedisConfig               `yaml:"redis"`
	Logger              cmd.LoggerConfig              `yaml:"log"`
	API                 apiConfig                     `yaml:"api"`
	Web                 webConfig                     `yaml:"web"`
	Telemetry           cmd.TelemetryConfig           `yaml:"telemetry"`
	Remotes             cmd.RemotesConfig             `yaml:",inline"`
	NotificationHistory cmd.NotificationHistoryConfig `yaml:"notification_history"`
	Checks              cmd.ChecksConfig              `yaml:"selfstate_checks"`
}

func toCheckConfig(checksConfig cmd.ChecksConfig) *selfstate.ChecksConfig {
	return &selfstate.ChecksConfig{
		Database: selfstate.HeartbeatConfig{
			SystemTags: checksConfig.Database.SystemTags,
		},
		Filter: selfstate.HeartbeatConfig{
			SystemTags: checksConfig.Filter.SystemTags,
		},
		LocalChecker: selfstate.HeartbeatConfig{
			SystemTags: checksConfig.LocalChecker.SystemTags,
		},
		RemoteChecker: selfstate.HeartbeatConfig{
			SystemTags: checksConfig.RemoteChecker.SystemTags,
		},
		Notifier: selfstate.NotifierHeartbeatConfig{
			DefaultTags:     checksConfig.Notifier.DefaultTags,
			LocalSourceTags: checksConfig.Notifier.LocalSourceTags,
			SourceTagPrefix: checksConfig.Notifier.SourceTagPrefix,
		},
	}
}

// ClustersMetricTTL parses TTLs of all clusters provided in config.
func (config *config) ClustersMetricTTL() map[moira.ClusterKey]time.Duration {
	result := make(map[moira.ClusterKey]time.Duration)

	result[moira.DefaultLocalCluster] = to.Duration(config.Redis.MetricsTTL)

	for _, remote := range config.Remotes.Graphite {
		key := moira.MakeClusterKey(moira.GraphiteRemote, remote.ClusterId)
		result[key] = to.Duration(remote.MetricsTTL)
	}

	for _, remote := range config.Remotes.Prometheus {
		key := moira.MakeClusterKey(moira.PrometheusRemote, remote.ClusterId)
		result[key] = to.Duration(remote.MetricsTTL)
	}

	return result
}

type apiConfig struct {
	// Api local network address. Default is ':8081' so api will be available at http://moira.company.com:8081/api.
	Listen string `yaml:"listen"`
	// If true, CORS for cross-domain requests will be enabled. This option can be used only for debugging purposes.
	EnableCORS bool `yaml:"enable_cors"`
	// Authorization contains authorization configuration.
	Authorization authorization `yaml:"authorization"`
	// Limits contains limits applied to entities and so on.
	Limits LimitsConfig `yaml:"limits"`
}

// LimitsConfig contains configurable moira limits.
type LimitsConfig struct {
	// Pager contains the limits applied to pagers.
	Pager PagerLimits `yaml:"pager"`
	// Trigger contains the limits applied to triggers.
	Trigger TriggerLimitsConfig `yaml:"trigger"`
	// Team contains the limits applied to teams.
	Team TeamLimitsConfig `yaml:"team"`
	// Contact contains the limits applied to contacts.
	Contact ContactLimits `yaml:"contact"`
}

// PagerLimits represents the limits which will be applied to all pagers.
type PagerLimits struct {
	// TTL is the amount of time that the pager will be exist
	TTL time.Duration `yaml:"ttl"`
}

// TriggerLimitsConfig represents the limits which will be applied to all triggers.
type TriggerLimitsConfig struct {
	// MaxNameSize is the max amount of characters allowed in trigger name.
	MaxNameSize int `yaml:"max_name_size"`
}

// TeamLimitsConfig represents the limits which will be applied to all teams.
type TeamLimitsConfig struct {
	// MaxNameSize is the max amount of characters allowed in team name.
	MaxNameSize int `yaml:"max_name_size"`
	// MaxDescriptionSize is the max amount of characters allowed in team description.
	MaxDescriptionSize int `yaml:"max_description_size"`
}

// ContactLimits defines limits for contact-related configurations.
type ContactLimits struct {
	// TestNotificationWaitTime specifies the duration to wait for test notifications.
	TestNotificationWaitTime time.Duration `yaml:"test_notification_wait_time"`
}

// ToLimits converts LimitsConfig to api.LimitsConfig.
func (conf LimitsConfig) ToLimits() api.LimitsConfig {
	return api.LimitsConfig{
		Pager: api.PagerLimits{
			TTL: conf.Pager.TTL,
		},
		Trigger: api.TriggerLimits{
			MaxNameSize: conf.Trigger.MaxNameSize,
		},
		Team: api.TeamLimits{
			MaxNameSize:        conf.Team.MaxNameSize,
			MaxDescriptionSize: conf.Team.MaxDescriptionSize,
		},
		Contact: api.ContactLimits{
			TestNotificationWaitTime: conf.Contact.TestNotificationWaitTime,
		},
	}
}

type authorization struct {
	// True if should limit non-admins and give admins additional privileges.
	Enabled bool `yaml:"enabled"`
	// List of logins of users who are considered to be admins.
	AdminList []string `yaml:"admin_list"`
	// List for control trigger deletion and editing, if createdBy fit in this list: only who create trigger can delete it.
	LimitedChangeTriggerOwners []string `yaml:"limited_change_trigger_owners"`
}

type sentryConfig struct {
	DSN      string `yaml:"dsn"`
	Platform string `yaml:"platform"`
}

func (config *sentryConfig) getSettings() api.Sentry {
	return api.Sentry{
		DSN:      config.DSN,
		Platform: config.Platform,
	}
}

type webConfig struct {
	// Moira administrator email address.
	SupportEmail string `yaml:"supportEmail"`
	// If true, users will be able to choose Graphite as trigger metrics data source.
	RemoteAllowed bool
	// List of enabled contacts template.
	ContactsTemplate []webContact `yaml:"contacts_template"`
	// Struct to manage feature flags.
	FeatureFlags featureFlags `yaml:"feature_flags"`
	// Returns the sentry configuration for the frontend.
	Sentry sentryConfig `yaml:"sentry"`
}

type webContact struct {
	// Contact type. Use sender name for script and webhook senders, in other cases use sender type.
	// See senders section of notifier config for more details: https://moira.readthedocs.io/en/latest/installation/configuration.html#notifier.
	ContactType string `yaml:"type"`
	// Contact type label that will be shown in web ui.
	ContactLabel string `yaml:"label"`
	// Logo URI sets the uri to the image with the contact's logo.
	LogoURI string `yaml:"logo_uri"`
	// Regular expression to match valid contact values.
	ValidationRegex string `yaml:"validation"`
	// Short description/example of valid contact value.
	Placeholder string `yaml:"placeholder"`
	// More detailed contact description.
	Help string `yaml:"help"`
}

type featureFlags struct {
	IsPlottingDefaultOn              bool   `yaml:"is_plotting_default_on"`
	IsPlottingAvailable              bool   `yaml:"is_plotting_available"`
	IsSubscriptionToAllTagsAvailable bool   `yaml:"is_subscription_to_all_tags_available"`
	IsReadonlyEnabled                bool   `yaml:"is_readonly_enabled"`
	CelebrationMode                  string `yaml:"celebration_mode"`
}

func (config *apiConfig) getSettings(
	metricsTTL map[moira.ClusterKey]time.Duration,
	flags api.FeatureFlags,
	webConfig *webConfig,
) *api.Config {
	return &api.Config{
		EnableCORS:    config.EnableCORS,
		Listen:        config.Listen,
		MetricsTTL:    metricsTTL,
		Flags:         flags,
		Authorization: config.Authorization.toApiConfig(webConfig),
		Limits:        config.Limits.ToLimits(),
	}
}

func (auth *authorization) toApiConfig(webConfig *webConfig) api.Authorization {
	adminList := make(map[string]struct{}, len(auth.AdminList))
	for _, admin := range auth.AdminList {
		adminList[admin] = struct{}{}
	}

	allowedContactTypes := make(map[string]struct{}, len(webConfig.ContactsTemplate))

	for _, contactTemplate := range webConfig.ContactsTemplate {
		allowedContactTypes[contactTemplate.ContactType] = struct{}{}
	}

	canChangeTriggersList := make(map[string]struct{}, len(auth.LimitedChangeTriggerOwners))
	for _, login := range auth.LimitedChangeTriggerOwners {
		canChangeTriggersList[login] = struct{}{}
	}

	return api.Authorization{
		Enabled:                    auth.Enabled,
		AdminList:                  adminList,
		AllowedContactTypes:        allowedContactTypes,
		LimitedChangeTriggerOwners: canChangeTriggersList,
	}
}

func (config *webConfig) validate() error {
	for _, contactTemplate := range config.ContactsTemplate {
		validationRegex := contactTemplate.ValidationRegex
		if validationRegex == "" {
			continue
		}

		if _, err := regexp.Compile(validationRegex); err != nil {
			return fmt.Errorf("contact template regex error '%s': %w", validationRegex, err)
		}
	}

	return nil
}

func (config *webConfig) getSettings(isRemoteEnabled bool, remotes cmd.RemotesConfig, redisConfig cmd.RedisConfig) *api.WebConfig {
	webContacts := make([]api.WebContact, 0, len(config.ContactsTemplate))

	for _, contactTemplate := range config.ContactsTemplate {
		contact := api.WebContact{
			ContactType:     contactTemplate.ContactType,
			ContactLabel:    contactTemplate.ContactLabel,
			LogoURI:         contactTemplate.LogoURI,
			ValidationRegex: contactTemplate.ValidationRegex,
			Placeholder:     contactTemplate.Placeholder,
			Help:            contactTemplate.Help,
		}

		webContacts = append(webContacts, contact)
	}

	clusters := cmd.MakeClustersWebConfig(redisConfig, remotes)

	return &api.WebConfig{
		SupportEmail:         config.SupportEmail,
		RemoteAllowed:        isRemoteEnabled,
		MetricSourceClusters: clusters,
		Contacts:             webContacts,
		FeatureFlags:         config.getFeatureFlags(),
		Sentry:               config.Sentry.getSettings(),
	}
}

func (config *webConfig) getFeatureFlags() api.FeatureFlags {
	return api.FeatureFlags{
		IsPlottingDefaultOn:              config.FeatureFlags.IsPlottingDefaultOn,
		IsPlottingAvailable:              config.FeatureFlags.IsPlottingAvailable,
		IsSubscriptionToAllTagsAvailable: config.FeatureFlags.IsSubscriptionToAllTagsAvailable,
		IsReadonlyEnabled:                config.FeatureFlags.IsReadonlyEnabled,
		CelebrationMode:                  getCelebrationMode(config.FeatureFlags.CelebrationMode),
	}
}

func getCelebrationMode(mode string) api.CelebrationMode {
	celebrationMode := api.CelebrationMode(mode)
	if api.IsAvailableCelebrationMode(celebrationMode) {
		return celebrationMode
	}

	return ""
}

func getDefault() config {
	return config{
		Redis: cmd.DefaultRedisConfig(),
		NotificationHistory: cmd.NotificationHistoryConfig{
			NotificationHistoryTTL: "48h",
		},
		Logger: cmd.LoggerConfig{
			LogFile:         "stdout",
			LogLevel:        "info",
			LogPrettyFormat: false,
		},
		API: apiConfig{
			Listen:     ":8081",
			EnableCORS: false,
			Limits: LimitsConfig{
				Pager: PagerLimits{
					TTL: api.DefaultTriggerPagerTTL,
				},
				Trigger: TriggerLimitsConfig{
					MaxNameSize: api.DefaultTriggerNameMaxSize,
				},
				Team: TeamLimitsConfig{
					MaxNameSize:        api.DefaultTeamNameMaxSize,
					MaxDescriptionSize: api.DefaultTeamDescriptionMaxSize,
				},
				Contact: ContactLimits{
					TestNotificationWaitTime: 10 * time.Second,
				},
			},
		},
		Web: webConfig{
			RemoteAllowed: false,
			FeatureFlags: featureFlags{
				IsPlottingDefaultOn:              true,
				IsPlottingAvailable:              true,
				IsSubscriptionToAllTagsAvailable: true,
			},
		},
		Telemetry: cmd.TelemetryConfig{
			Listen: ":8091",
			Graphite: cmd.GraphiteConfig{
				Enabled:      false,
				RuntimeStats: false,
				URI:          "localhost:2003",
				Prefix:       "DevOps.Moira",
				Interval:     "60s",
			},
			Pprof: cmd.ProfilerConfig{Enabled: false},
		},
		Remotes: cmd.RemotesConfig{},
	}
}
