package main

import (
	"fmt"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/cmd"
)

type config struct {
	Redis    cmd.RedisConfig    `yaml:"redis"`
	Graphite cmd.GraphiteConfig `yaml:"graphite"`
	Logger   cmd.LoggerConfig   `yaml:"log"`
	API      apiConfig          `yaml:"api"`
	Web      webConfig          `yaml:"web"`
	Pprof    cmd.ProfilerConfig `yaml:"pprof"`
	Remote   cmd.RemoteConfig   `yaml:"remote"`
}

type apiConfig struct {
	// Api local network address. Default is ':8081' so api will be available at http://moira.company.com:8081/api
	Listen string `yaml:"listen"`
	// If true, CORS for cross-domain requests will be enabled. This option can be used only for debugging purposes.
	EnableCORS bool `yaml:"enable_cors"`
}

type webConfig struct {
	SupportEmail  string       `yaml:"supportEmail"`
	RemoteAllowed bool         `yaml:"remoteAllowed"`
	Contacts      []webContact `yaml:"contacts"`
}

type webContact struct {
	ContactType     string `yaml:"type"`
	ContactLabel    string `yaml:"label"`
	ValidationRegex string `yaml:"validation"`
	Placeholder     string `yaml:"placeholder"`
}

func (config *apiConfig) getSettings() *api.Config {
	return &api.Config{
		Listen:     config.Listen,
		EnableCORS: config.EnableCORS,
	}
}

func (config *webConfig) getSettings(isRemoteEnabled bool) (*api.WebConfig, error) {
	if !isRemoteEnabled && config.RemoteAllowed {
		return nil, fmt.Errorf("to allow usage of remote triggers, remote: enabled must be set to true")
	}
	webContacts := make([]api.WebContact, 0, len(config.Contacts))
	for _, configContact := range config.Contacts {
		contact := api.WebContact{
			ContactType:     configContact.ContactType,
			ContactLabel:    configContact.ContactLabel,
			ValidationRegex: configContact.ValidationRegex,
			Placeholder:     configContact.Placeholder,
		}
		webContacts = append(webContacts, contact)
	}
	return &api.WebConfig{
		SupportEmail:  config.SupportEmail,
		RemoteAllowed: config.RemoteAllowed,
		Contacts:      webContacts,
	}, nil
}

func getDefault() config {
	return config{
		Redis: cmd.RedisConfig{
			Host:            "localhost",
			Port:            "6379",
			ConnectionLimit: 512,
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
		Graphite: cmd.GraphiteConfig{
			RuntimeStats: false,
			URI:          "localhost:2003",
			Prefix:       "DevOps.Moira",
			Interval:     "60s",
		},
		Pprof: cmd.ProfilerConfig{
			Listen: "",
		},
		Remote: cmd.RemoteConfig{
			Timeout: "60s",
		},
	}
}
