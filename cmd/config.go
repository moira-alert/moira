package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/moira-alert/moira/metrics"

	"github.com/gosexy/to"
	"github.com/moira-alert/moira/image_store/s3"
	remoteSource "github.com/moira-alert/moira/metric_source/remote"
	"gopkg.in/yaml.v2"

	"github.com/moira-alert/moira/database/redis"
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
	// Redis database
	DB              int `yaml:"dbid"`
	ConnectionLimit int `yaml:"connection_limit"`
}

// GetSettings returns redis config parsed from moira config files
func (config *RedisConfig) GetSettings() redis.Config {
	return redis.Config{
		MasterName:        config.MasterName,
		SentinelAddresses: strings.Split(config.SentinelAddrs, ","),
		Host:              config.Host,
		Port:              config.Port,
		DB:                config.DB,
		ConnectionLimit:   config.ConnectionLimit,
	}
}

// GraphiteConfig is graphite metrics config structure that initialises at the start of moira
type GraphiteConfig struct {
	// If true, graphite sender will be enabled.
	Enabled bool `yaml:"enabled"`
	// If true, runtime stats will be captured and sent to graphite. Note: It takes to call stoptheworld() with configured "graphite.interval" to capture runtime stats (https://golang.org/src/runtime/mstats.go)
	RuntimeStats bool `yaml:"runtime_stats"`
	// Graphite relay URI, format: ip:port
	URI string `yaml:"uri"`
	// Moira metrics prefix. Use 'prefix: {hostname}' to use hostname autoresolver.
	Prefix string `yaml:"prefix"`
	// Metrics sending interval
	Interval string `yaml:"interval"`
}

// GetSettings returns graphite metrics config parsed from moira config files
func (graphiteConfig *GraphiteConfig) GetSettings() metrics.GraphiteRegistryConfig {
	return metrics.GraphiteRegistryConfig{
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

// TelemetryConfig is settings for listener, pprof, graphite
type TelemetryConfig struct {
	Listen   string         `yaml:"listen"`
	Pprof    ProfilerConfig `yaml:"pprof"`
	Graphite GraphiteConfig `yaml:"graphite"`
}

// ProfilerConfig is pprof settings structure that initialises at the start of moira
type ProfilerConfig struct {
	Enabled bool `yaml:"enabled"`
}

// RemoteConfig is remote graphite settings structure
type RemoteConfig struct {
	// graphite url e.g http://graphite/render
	URL string `yaml:"url"`
	// Min period to perform triggers re-check. Note: Reducing of this value leads to increasing of CPU and memory usage values
	CheckInterval string `yaml:"check_interval"`
	// Timeout for remote requests
	Timeout string `yaml:"timeout"`
	// Username for basic auth
	User string `yaml:"user"`
	// Password for basic auth
	Password string `yaml:"password"`
	// If true, remote worker will be enabled.
	Enabled bool `yaml:"enabled"`
}

// ImageStoreConfig defines the configuration for all the image stores to be initialized by InitImageStores
type ImageStoreConfig struct {
	S3 s3.Config `yaml:"s3"`
}

// GetRemoteSourceSettings returns remote config parsed from moira config files
func (config *RemoteConfig) GetRemoteSourceSettings() *remoteSource.Config {
	return &remoteSource.Config{
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
