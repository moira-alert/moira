package main

import (
	"encoding/json"
	"fmt"

	"github.com/xiam/to"

	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/cmd"
)

type config struct {
	Redis     cmd.RedisConfig     `yaml:"redis"`
	Logger    cmd.LoggerConfig    `yaml:"log"`
	API       apiConfig           `yaml:"api"`
	Web       webConfig           `yaml:"web"`
	Telemetry cmd.TelemetryConfig `yaml:"telemetry"`
	Remote    cmd.RemoteConfig    `yaml:"remote"`
}

type apiConfig struct {
	// Api local network address. Default is ':8081' so api will be available at http://moira.company.com:8081/api.
	Listen string `yaml:"listen"`
	// If true, CORS for cross-domain requests will be enabled. This option can be used only for debugging purposes.
	EnableCORS bool `yaml:"enable_cors"`
}

type webConfig struct {
	// Moira administrator email address.
	SupportEmail string `yaml:"supportEmail"`
	// If true, users will be able to choose Graphite as trigger metrics data source
	RemoteAllowed bool
	// List of enabled contact types
	Contacts []webContact `yaml:"contacts"`
}

type webContact struct {
	// Contact type. Use sender name for script and webhook senders, in other cases use sender type.
	// See senders section of notifier config for more details: https://moira.readthedocs.io/en/latest/installation/configuration.html#notifier
	ContactType string `yaml:"type"`
	// Contact type label that will be shown in web ui
	ContactLabel string `yaml:"label"`
	// Regular expression to match valid contact values
	ValidationRegex string `yaml:"validation"`
	// Short description/example of valid contact value
	Placeholder string `yaml:"placeholder"`
	// More detailed contact description
	Help string `yaml:"help"`
}

func (config *apiConfig) getSettings(localMetricTTL, remoteMetricTTL string) *api.Config {
	return &api.Config{
		EnableCORS:      config.EnableCORS,
		Listen:          config.Listen,
		LocalMetricTTL:  to.Duration(localMetricTTL),
		RemoteMetricTTL: to.Duration(remoteMetricTTL),
	}
}

func (config *webConfig) getSettings(isRemoteEnabled bool) ([]byte, error) {
	webContacts := make([]api.WebContact, 0, len(config.Contacts))
	for _, configContact := range config.Contacts {
		contact := api.WebContact{
			ContactType:     configContact.ContactType,
			ContactLabel:    configContact.ContactLabel,
			ValidationRegex: configContact.ValidationRegex,
			Placeholder:     configContact.Placeholder,
			Help:            configContact.Help,
		}
		webContacts = append(webContacts, contact)
	}
	configContent, err := json.Marshal(api.WebConfig{
		SupportEmail:  config.SupportEmail,
		RemoteAllowed: isRemoteEnabled,
		Contacts:      webContacts,
	})
	if err != nil {
		return make([]byte, 0), fmt.Errorf("failed to parse web config: %s", err.Error())
	}
	return configContent, nil
}

func getDefault() config {
	return config{
		Redis: cmd.RedisConfig{
			Host:            "localhost",
			Port:            "6379",
			ConnectionLimit: 512, //nolint
			MetricsTTL:      "1h",
		},
		Logger: cmd.LoggerConfig{
			LogFile:  "stdout",
			LogLevel: "info",
		},
		API: apiConfig{
			Listen:     ":8081",
			EnableCORS: false,
		},
		Web: webConfig{
			RemoteAllowed: false,
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
		Remote: cmd.RemoteConfig{
			Timeout:    "60s",
			MetricsTTL: "7d",
		},
	}
}
