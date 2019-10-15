package main

import (
	"github.com/moira-alert/moira/cmd"
)

type config struct {
	LogFile  string          `yaml:"log_file"`
	LogLevel string          `yaml:"log_level"`
	Redis    cmd.RedisConfig `yaml:"redis"`
}

func getDefault() config {
	return config{
		LogFile:  "stdout",
		LogLevel: "info",
		Redis: cmd.RedisConfig{
			Host:            "redis",
			Port:            "6379",
			ConnectionLimit: 512,
		},
	}
}
