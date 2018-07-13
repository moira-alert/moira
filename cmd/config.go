package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/gosexy/to"
	"gopkg.in/yaml.v2"

	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/metrics/graphite"
)

// RedisConfig is a redis config structure that initialises at the start of moira
// Use fields MasterName and SentinelAddrs to enable Redis Sentinel support,
// use Host and Port fields otherwise.
type RedisConfig struct {
	// Redis Sentinel cluster name
	MasterName string `yaml:"master_name"`
	// Redis Sentinel address list, format: {host1_name:port};{ip:port}
	SentinelAddrs string `yaml:"sentinel_addrs"`
	// Redis node ip-address or host name
	Host string `yaml:"host"`
	// Redis node port
	Port string `yaml:"port"`
	// Redis database id
	DBID int `yaml:"dbid"`
}

// GetSettings returns redis config parsed from moira config files
func (config *RedisConfig) GetSettings() redis.Config {
	return redis.Config{
		MasterName:        config.MasterName,
		SentinelAddresses: strings.Split(config.SentinelAddrs, ","),
		Host:              config.Host,
		Port:              config.Port,
		DBID:              config.DBID,
	}
}

// GraphiteConfig is graphite metrics config structure that initialises at the start of moira
type GraphiteConfig struct {
	// If true, graphite logger will be enabled.
	Enabled bool `yaml:"enabled"`
	// Graphite relay URI, format: ip:port
	URI string `yaml:"uri"`
	// Moira metrics prefix. Use 'prefix: {hostname}' to use hostname autoresolver.
	Prefix string `yaml:"prefix"`
	// Metrics sending interval
	Interval string `yaml:"interval"`
}

// GetSettings returns graphite metrics config parsed from moira config files
func (graphiteConfig *GraphiteConfig) GetSettings() graphite.Config {
	return graphite.Config{
		Enabled:  graphiteConfig.Enabled,
		URI:      graphiteConfig.URI,
		Prefix:   graphiteConfig.Prefix,
		Interval: to.Duration(graphiteConfig.Interval),
	}
}

// LoggerConfig is logger settings structure that initialises at the start of moira
type LoggerConfig struct {
	LogFile  string `yaml:"log_file"`
	LogLevel string `yaml:"log_level"`
}

// ProfilerConfig is pprof settings structure that initialises at the start of moira
type ProfilerConfig struct {
	// Define variable as valid non-empty string to enable pprof server. For example ':10000' will enable server available at http://moira.company.com:10000/debug/pprof/
	Listen string `yaml:"listen"`
}

// ReadConfig parses config file by the given path into Moira-used type
func ReadConfig(configFileName string, config interface{}) error {
	configYaml, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return fmt.Errorf("can't read file [%s] [%s]", configFileName, err.Error())
	}
	err = yaml.Unmarshal(configYaml, config)
	if err != nil {
		return fmt.Errorf("can't parse config file [%s] [%s]", configFileName, err.Error())
	}
	return nil
}

// PrintConfig prints config to stdout
func PrintConfig(config interface{}) {
	d, _ := yaml.Marshal(&config)
	fmt.Println(string(d))
}
