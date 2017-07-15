package main

import (
	"fmt"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type config struct {
	Redis    redisConfig    `yaml:"redis"`
	Cache    cacheConfig    `yaml:"cache"`
	Graphite graphiteConfig `yaml:"graphite"`
}

type cacheConfig struct {
	LogLevel        string `yaml:"log_level"`
	LogColor        string `yaml:"log_color"`
	LogFile         string `yaml:"log_file"`
	PidFile         string `yaml:"pid"`
	Listen          string `yaml:"listen"`
	RetentionConfig string `yaml:"retention-config"`
}

type redisConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	DBID int    `yaml:"dbid"`
}

type graphiteConfig struct {
	URI      string `yaml:"uri"`
	Prefix   string `yaml:"prefix"`
	Interval int64  `yaml:"interval"`
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

func getDefault() config {
	return config{
		Redis: redisConfig{
			Host: "localhost",
			Port: "6379",
		},
		Cache: cacheConfig{
			PidFile:         "",
			LogFile:         "stdout",
			Listen:          "",
			RetentionConfig: "",
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
