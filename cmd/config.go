package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
	"menteslibres.net/gosexy/to"

	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/metrics/graphite"
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
	Enabled  string `yaml:"enabled"`
	URI      string `yaml:"uri"`
	Prefix   string `yaml:"prefix"`
	Interval string `yaml:"interval"`
}

func (graphiteConfig *GraphiteConfig) GetSettings() graphite.Config {
	return graphite.Config{
		Enabled:  ToBool(graphiteConfig.Enabled),
		URI:      graphiteConfig.URI,
		Prefix:   graphiteConfig.Prefix,
		Interval: to.Duration(graphiteConfig.Interval),
	}
}

type LoggerConfig struct {
	LogFile  string `yaml:"log_file"`
	LogLevel string `yaml:"log_level"`
}
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

func PrintConfig(config interface{}) {
	d, _ := yaml.Marshal(&config)
	fmt.Println(string(d))
}

func ToBool(str string) bool {
	switch strings.ToLower(str) {
	case "1", "true", "t", "yes", "y":
		return true
	}
	return false
}
