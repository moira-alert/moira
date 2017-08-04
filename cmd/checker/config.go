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
	Checker  checkerConfig  `yaml:"checker"`
	Graphite graphiteConfig `yaml:"graphite"`
	Front    frontConfig    `yaml:"front"`
}

type redisConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	DBID int    `yaml:"dbid"`
}

type checkerConfig struct {
	LogFile              string `yaml:"log_file"`
	LogLevel             string `yaml:"log_level"`
	LogColor             string `yaml:"log_color"`
	NoDataCheckInterval  int64  `yaml:"nodata_check_interval"`
	CheckInterval        int64  `yaml:"check_interval"`
	MetricsTTL           int64  `yaml:"metrics_ttl"`
	StopCheckingInterval int64  `yaml:"stop_checking_interval"`
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

func (config *checkerConfig) getLoggerSettings(verbosityLog *bool) logging.Config {
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
		Checker: checkerConfig{
			LogFile:              "stdout",
			NoDataCheckInterval:  60,
			CheckInterval:        5,
			MetricsTTL:           3600,
			StopCheckingInterval: 30,
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
