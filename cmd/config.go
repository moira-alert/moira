package cmd

import (
	"fmt"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/logging"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"menteslibres.net/gosexy/to"
	"strings"
)

type RedisConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	DBID int    `yaml:"dbid"`
}

func (config *RedisConfig) GetSettings() redis.Config {
	return redis.Config{
		Host: config.Host,
		Port: config.Port,
		DBID: config.DBID,
	}
}

type GraphiteConfig struct {
	URI      string `yaml:"uri"`
	Prefix   string `yaml:"prefix"`
	Interval string `yaml:"interval"`
}

func (graphiteConfig *GraphiteConfig) GetSettings() graphite.Config {
	return graphite.Config{
		URI:      graphiteConfig.URI,
		Prefix:   graphiteConfig.Prefix,
		Interval: to.Duration(graphiteConfig.Interval),
	}
}

type LoggerConfig struct {
	LogFile  string `yaml:"log_file"`
	LogLevel string `yaml:"log_level"`
	LogColor string `yaml:"log_color"`
}

func (loggerConfig *LoggerConfig) GetSettings(verbosityLog bool) logging.Config {
	cfg := logging.Config{
		LogFile:  loggerConfig.LogFile,
		LogLevel: loggerConfig.LogLevel,
		LogColor: toBool(loggerConfig.LogColor),
	}
	if verbosityLog {
		cfg.LogLevel = "debug"
	}
	return cfg
}

func ReadConfig(configFileName string, config interface{}) error {
	configYaml, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return fmt.Errorf("Can't read file [%s] [%s]", configFileName, err.Error())
	}
	err = yaml.Unmarshal(configYaml, &config)
	if err != nil {
		return fmt.Errorf("Can't parse config file [%s] [%s]", configFileName, err.Error())
	}
	return nil
}

func PrintConfig(config interface{}) {
	d, _ := yaml.Marshal(&config)
	fmt.Println(string(d))
}

func toBool(str string) bool {
	switch strings.ToLower(str) {
	case "1", "true", "t", "yes", "y":
		return true
	}
	return false
}
