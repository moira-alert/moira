package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
	"menteslibres.net/gosexy/to"

	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/metrics/graphite"
)

// RedisConfig is redis config structure, which are taken on the start of moira
type RedisConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	DBID int    `yaml:"dbid"`
}

// GetSettings return redis config parsed from moira config files
func (config *RedisConfig) GetSettings() redis.Config {
	return redis.Config{
		Host: config.Host,
		Port: config.Port,
		DBID: config.DBID,
	}
}

// GraphiteConfig is graphite metrics config, which are taken on the start of moira
type GraphiteConfig struct {
	Enabled  string `yaml:"enabled"`
	URI      string `yaml:"uri"`
	Prefix   string `yaml:"prefix"`
	Interval string `yaml:"interval"`
}

// GetSettings return graphite metrics config parsed from moira config files
func (graphiteConfig *GraphiteConfig) GetSettings() graphite.Config {
	return graphite.Config{
		Enabled:  ToBool(graphiteConfig.Enabled),
		URI:      graphiteConfig.URI,
		Prefix:   graphiteConfig.Prefix,
		Interval: to.Duration(graphiteConfig.Interval),
	}
}

// LoggerConfig is logger settings, which are taken on the start of moira
type LoggerConfig struct {
	LogFile  string `yaml:"log_file"`
	LogLevel string `yaml:"log_level"`
}

// ProfilerConfig is pprof settings, which are taken on the start of moira
type ProfilerConfig struct {
	Port string `yaml:"port"`
}

// ReadConfig gets config file by given file and marshal it to moira-used type
func ReadConfig(configFileName string, config interface{}) error {
	configYaml, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return fmt.Errorf("Can't read file [%s] [%s]", configFileName, err.Error())
	}
	err = yaml.Unmarshal(configYaml, config)
	if err != nil {
		return fmt.Errorf("Can't parse config file [%s] [%s]", configFileName, err.Error())
	}
	return nil
}

// PrintConfig prints config to std
func PrintConfig(config interface{}) {
	d, _ := yaml.Marshal(&config)
	fmt.Println(string(d))
}

// ToBool is simple func witch parse popular bool interpretation to golang bool value
func ToBool(str string) bool {
	switch strings.ToLower(str) {
	case "1", "true", "t", "yes", "y":
		return true
	}
	return false
}
