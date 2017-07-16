package main

import (
	"fmt"
	"github.com/gosexy/to"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/logging"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"github.com/moira-alert/moira-alert/notifier"
	"github.com/moira-alert/moira-alert/notifier/selfstate"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

type config struct {
	Redis    redisConfig    `yaml:"redis"`
	Front    frontConfig    `yaml:"front"`
	Graphite graphiteConfig `yaml:"graphite"`
	Notifier notifierConfig `yaml:"notifier"`
}

type notifierConfig struct {
	LogFile          string              `yaml:"log_file"`
	LogLevel         string              `yaml:"log_level"`
	LogColor         string              `yaml:"log_color"`
	SenderTimeout    string              `yaml:"sender_timeout"`
	ResendingTimeout string              `yaml:"resending_timeout"`
	Senders          []map[string]string `yaml:"senders"`
	SelfState        selfStateConfig     `yaml:"moira_selfstate"`
}

type redisConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	DBID int    `yaml:"dbid"`
}

type frontConfig struct {
	URI string `yaml:"uri"`
}

type graphiteConfig struct {
	URI      string `yaml:"uri"`
	Prefix   string `yaml:"prefix"`
	Interval int64  `yaml:"interval"`
}

type selfStateConfig struct {
	Enabled                 string              `yaml:"enabled"`
	RedisDisconnectDelay    int64               `yaml:"redis_disconect_delay"`
	LastMetricReceivedDelay int64               `yaml:"last_metric_received_delay"`
	LastCheckDelay          int64               `yaml:"last_check_delay"`
	Contacts                []map[string]string `yaml:"contacts"`
	NoticeInterval          int64               `yaml:"notice_interval"`
}

func getDefault() config {
	return config{
		Redis: redisConfig{
			Host: "localhost",
			Port: "6379",
		},
		Front: frontConfig{
			URI: "http://localhost",
		},
		Graphite: graphiteConfig{
			URI:      "localhost:2003",
			Prefix:   "DevOps.Moira",
			Interval: 60,
		},
		Notifier: notifierConfig{
			LogFile:          "stdout",
			SenderTimeout:    "10s0ms",
			ResendingTimeout: "24:00",
			SelfState: selfStateConfig{
				Enabled:                 "false",
				RedisDisconnectDelay:    30,
				LastMetricReceivedDelay: 60,
				LastCheckDelay:          60,
				NoticeInterval:          300,
			},
		},
	}
}

func (graphiteConfig *graphiteConfig) getSettings() graphite.Config {
	return graphite.Config{
		URI:      graphiteConfig.URI,
		Prefix:   graphiteConfig.Prefix,
		Interval: graphiteConfig.Interval,
	}
}

func (config *redisConfig) getSettings() redis.Config {
	return redis.Config{
		Host: config.Host,
		Port: config.Port,
		DBID: config.DBID,
	}
}

func (config *notifierConfig) getSettings() notifier.Config {
	return notifier.Config{
		SendingTimeout:   to.Duration(config.SenderTimeout),
		ResendingTimeout: to.Duration(config.ResendingTimeout),
		Senders:          config.Senders,
	}
}

func (config *notifierConfig) getLoggerSettings() logging.Config {
	return logging.Config{
		LogFile:  config.LogFile,
		LogColor: toBool(config.LogColor),
		LogLevel: config.LogLevel,
	}
}

func (config *selfStateConfig) getSettings() selfstate.Config {
	return selfstate.Config{
		Enabled:                 toBool(config.Enabled),
		RedisDisconnectDelay:    config.RedisDisconnectDelay,
		LastMetricReceivedDelay: config.LastMetricReceivedDelay,
		LastCheckDelay:          config.LastCheckDelay,
		Contacts:                config.Contacts,
		NoticeInterval:          config.NoticeInterval,
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
