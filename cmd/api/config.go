package main

import (
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/cmd"
)

type config struct {
	Redis  cmd.RedisConfig    `yaml:"redis"`
	Logger cmd.LoggerConfig   `yaml:"log"`
	API    apiConfig          `yaml:"api"`
	Pprof  cmd.ProfilerConfig `yaml:"pprof"`
}

type apiConfig struct {
	// Api local network address. Default is ':8081' so api will be available at http://moira.company.com:8081/api
	Listen string `yaml:"listen"`
	// If true, CORS for cross-domain requests will be enabled. This option can be used only for debugging purposes.
	EnableCORS bool `yaml:"enable_cors"`
	// Web_UI config file path. If file not found, api will return 404 in response to "api/config"
	WebConfigPath string `yaml:"web_config_path"`
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
		Pprof: cmd.ProfilerConfig{
			Listen: "",
		},
	}
}
