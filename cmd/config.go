package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/gosexy/to"
	"github.com/moira-alert/moira/remote"
	"gopkg.in/yaml.v2"

	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/metrics/graphite"
)

// RedisConfig is a redis config structure that initialises at the start of moira
// Use fields MasterName and SentinelAddrs to enable Redis Sentinel support,
// use Host and Port fields otherwise.
type RedisConfig struct {
	MasterName    string `yaml:"master_name"`    // Redis Sentinel cluster name
	SentinelAddrs string `yaml:"sentinel_addrs"` // Redis Sentinel address list, format: {host1_name:port};{ip:port}
	Host          string `yaml:"host"`           // Redis node ip-address or host name
	Port          string `yaml:"port"`           // Redis node port
	DBID          int    `yaml:"dbid"`           // Redis database id
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
	Enabled      bool   `yaml:"enabled"`       // If true, graphite sender will be enabled.
	RuntimeStats bool   `yaml:"runtime_stats"` // If true, runtime stats will be captured and sent to graphite. Note: It takes to call stoptheworld() with configured "graphite.interval" to capture runtime stats (https://golang.org/src/runtime/mstats.go)
	URI          string `yaml:"uri"`           // Graphite relay URI, format: ip:port
	Prefix       string `yaml:"prefix"`        // Moira metrics prefix. Use 'prefix: {hostname}' to use hostname autoresolver.
	Interval     string `yaml:"interval"`      // Metrics sending interval
}

// GetSettings returns graphite metrics config parsed from moira config files
func (graphiteConfig *GraphiteConfig) GetSettings() graphite.Config {
	return graphite.Config{
		Enabled:      graphiteConfig.Enabled,
		RuntimeStats: graphiteConfig.RuntimeStats,
		URI:          graphiteConfig.URI,
		Prefix:       graphiteConfig.Prefix,
		Interval:     to.Duration(graphiteConfig.Interval),
	}
}

// LoggerConfig is logger settings structure that initialises at the start of moira
type LoggerConfig struct {
	LogFile  string `yaml:"log_file"`
	LogLevel string `yaml:"log_level"`
}

// ProfilerConfig is pprof settings structure that initialises at the start of moira
type ProfilerConfig struct {
	Listen string `yaml:"listen"` // Define variable as valid non-empty string to enable pprof server. For example ':10000' will enable server available at http://moira.company.com:10000/debug/pprof/
}

type RemoteConfig struct {
	URL           string `yaml:"url"`
	CheckInterval string `yaml:"check_interval"`
	Timeout       string `yaml:"timeout"`
	User          string `yaml:"user"`
	Password      string `yaml:"password"`
	Enabled       bool   `yaml:"enabled"`
}

// GetSettings returns redis config parsed from moira config files
func (config *RemoteConfig) GetSettings() *remote.Config {
	return &remote.Config{
		URL:           config.URL,
		CheckInterval: to.Duration(config.CheckInterval),
		Timeout:       to.Duration(config.Timeout),
		User:          config.User,
		Password:      config.Password,
		Enabled:       config.Enabled,
	}
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
