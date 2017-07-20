package main

import (
	"fmt"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/logging"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

type config struct {
	Redis    redisConfig    `yaml:"redis"`
	Api      apiConfig      `yaml:"api"`
	Graphite graphiteConfig `yaml:"graphite"`
	Front    frontConfig    `yaml:"front"`
}

type redisConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	DBID int    `yaml:"dbid"`
}

type apiConfig struct {
	LogFile  string `yaml:"log_file"`
	LogLevel string `yaml:"log_level"`
	LogColor string `yaml:"log_color"`
	Port     string `yaml:"port"`
	Address  string `yaml:"listen"`
}

type graphiteConfig struct {
	URI      string `yaml:"uri"`
	Prefix   string `yaml:"prefix"`
	Interval int64  `yaml:"interval"`
}

type frontConfig struct {
	URI string `yaml:"uri"`
}

func (config *redisConfig) getSettings() redis.Config {
	return redis.Config{
		Host: config.Host,
		Port: config.Port,
		DBID: config.DBID,
	}
}

func (config *apiConfig) getLoggerSettings(verbosityLog *bool) logging.Config {
	conf := logging.Config{
		LogFile:  config.LogFile,
		LogColor: toBool(config.LogColor),
		LogLevel: config.LogLevel,
	}
	if *verbosityLog {
		config.LogLevel = "debug"
	}
	return conf
}

func getDefault() config {
	return config{
		Redis: redisConfig{
			Host: "localhost",
			Port: "6379",
		},
		Api: apiConfig{
			LogFile: "stdout",
			Port:    "8081",
			Address: "0.0.0.0",
		},
		Graphite: graphiteConfig{
			URI:      "localhost:2003",
			Prefix:   "DevOps.Moira",
			Interval: 60,
		},
	}
}

func readSettings(configFileName string) (*config, error) {
	c := getDefault()
	configYaml, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return nil, fmt.Errorf("Can't read file [%s] [%s]", configFileName, err.Error())
	}
	err = yaml.Unmarshal(configYaml, &c)
	if err != nil {
		return nil, fmt.Errorf("Can't parse config file [%s] [%s]", configFileName, err.Error())
	}
	return &c, nil
}

func toBool(str string) bool {
	switch strings.ToLower(str) {
	case "1", "true", "t", "yes", "y":
		return true
	}
	return false
}
