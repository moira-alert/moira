package main

import (
	"github.com/moira-alert/moira-alert/cmd"
)

type config struct {
	Redis  cmd.RedisConfig  `yaml:"redis"`
	Logger cmd.LoggerConfig `yaml:"log"`
	Api    apiConfig        `yaml:"api"`
}

type apiConfig struct {
	Port    string `yaml:"port"`
	Address string `yaml:"listen"`
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
		Api: apiConfig{
			Port:    "8081",
			Address: "0.0.0.0",
		},
	}
}
