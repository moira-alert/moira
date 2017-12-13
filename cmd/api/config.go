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
	Listen        string `yaml:"listen"`
	EnableCORS    bool   `yaml:"enable_cors"`
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
			LogLevel: "debug",
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
