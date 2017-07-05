package main

import (
	"fmt"
	"github.com/gosexy/to"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"github.com/moira-alert/moira-alert/notifier"
	"github.com/moira-alert/moira-alert/notifier/selfstate"
	"github.com/moira-alert/moira-alert/database/redis"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

type Config struct {
	Redis    RedisConfig    `yaml:"redis"`
	Front    FrontConfig    `yaml:"front"`
	Graphite GraphiteConfig `yaml:"graphite"`
	Notifier NotifierConfig `yaml:"notifier"`
}

type NotifierConfig struct {
	LogFile          string              `yaml:"log_file"`
	LogLevel         string              `yaml:"log_level"`
	LogColor         string              `yaml:"log_color"`
	SenderTimeout    string              `yaml:"sender_timeout"`
	ResendingTimeout string              `yaml:"resending_timeout"`
	Senders          []map[string]string `yaml:"senders"`
	SelfState        SelfStateConfig     `yaml:"moira_selfstate"`
}

type RedisConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	DBID int    `yaml:"dbid"`
}

type FrontConfig struct {
	URI string `yaml:"uri"`
}

type GraphiteConfig struct {
	URI      string `yaml:"uri"`
	Prefix   string `yaml:"prefix"`
	Interval int64  `yaml:"interval"`
}

type SelfStateConfig struct {
	Enabled                 string              `yaml:"enabled"`
	RedisDisconnectDelay    int64               `yaml:"redis_disconect_delay"`
	LastMetricReceivedDelay int64               `yaml:"last_metric_received_delay"`
	LastCheckDelay          int64               `yaml:"last_check_delay"`
	Contacts                []map[string]string `yaml:"contacts"`
	NoticeInterval          int64               `yaml:"notice_interval"`
}

func getDefault() Config {
	return Config{
		Redis: RedisConfig{
			Host: "localhost",
			Port: "6379",
		},
		Front: FrontConfig{
			URI: "http://localhost",
		},
		Graphite: GraphiteConfig{
			URI:      "localhost:2003",
			Prefix:   "DevOps.Moira",
			Interval: 60,
		},
		Notifier: NotifierConfig{
			LogFile:          "stdout",
			SenderTimeout:    "10s0ms",
			ResendingTimeout: "24:00",
			SelfState: SelfStateConfig{
				Enabled:                 "false",
				RedisDisconnectDelay:    30,
				LastMetricReceivedDelay: 60,
				LastCheckDelay:          60,
				NoticeInterval:          300,
			},
		},
	}
}

func (graphiteConfig *GraphiteConfig) GetSettings() graphite.Config {
	return graphite.Config{
		URI:      graphiteConfig.URI,
		Prefix:   graphiteConfig.Prefix,
		Interval: graphiteConfig.Interval,
	}
}

func (config *RedisConfig) GetSettings() redis.Config {
	return redis.Config{
		Host: config.Host,
		Port: config.Port,
		DBID: config.DBID,
	}
}

func (config *NotifierConfig) GetSettings() notifier.Config {
	return notifier.Config{
		LogFile:          config.LogFile,
		LogLevel:         config.LogLevel,
		LogColor:         toBool(config.LogColor),
		SendingTimeout:   to.Duration(config.SenderTimeout),
		ResendingTimeout: to.Duration(config.ResendingTimeout),
		Senders:          config.Senders,
	}
}

func (config *SelfStateConfig) GetSettings() selfstate.Config {
	return selfstate.Config{
		Enabled:                 toBool(config.Enabled),
		RedisDisconnectDelay:    config.RedisDisconnectDelay,
		LastMetricReceivedDelay: config.LastMetricReceivedDelay,
		LastCheckDelay:          config.LastCheckDelay,
		Contacts:                config.Contacts,
		NoticeInterval:          config.NoticeInterval,
	}
}

func readSettings(configFileName string) (*Config, error) {
	c := getDefault()
	configYaml, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return nil, fmt.Errorf("Can't read file [%s] [%s]", configFileName, err.Error())
	}
	err = yaml.Unmarshal([]byte(configYaml), &c)
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
