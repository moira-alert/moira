package main

import (
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/cmd"
)

type config struct {
	Redis    cmd.RedisConfig    `yaml:"redis"`
	Graphite cmd.GraphiteConfig `yaml:"graphite"`
	Logger   cmd.LoggerConfig   `yaml:"log"`
	API      apiConfig          `yaml:"api"`
	Pprof    cmd.ProfilerConfig `yaml:"pprof"`
	Remote   cmd.RemoteConfig   `yaml:"remote"`
}

type apiConfig struct {
	Listen        string `yaml:"listen"`          // Api local network address. Default is ':8081' so api will be available at http://moira.company.com:8081/api
	EnableCORS    bool   `yaml:"enable_cors"`     // If true, CORS for cross-domain requests will be enabled. This option can be used only for debugging purposes.
	WebConfigPath string `yaml:"web_config_path"` // Web_UI config file path. If file not found, api will return 404 in response to "api/config"
}

func (config *apiConfig) getSettings() *api.Config {
	return &api.Config{
		Listen:     config.Listen,
		EnableCORS: config.EnableCORS,
	}
}

func getDefault() config {
	return config{
		Redis: cmd.RedisConfig{
			Host: "localhost",
			Port: "6379",
		},
		Logger: cmd.LoggerConfig{
			LogFile:  "stdout",
			LogLevel: "info",
		},
		API: apiConfig{
			Listen:        ":8081",
			WebConfigPath: "/etc/moira/web.json",
			EnableCORS:    false,
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
	}
}
