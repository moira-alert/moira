package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"

	"github.com/moira-alert/moira/image_store/s3"
	prometheusRemoteSource "github.com/moira-alert/moira/metric_source/prometheus"
	graphiteRemoteSource "github.com/moira-alert/moira/metric_source/remote"
	"github.com/xiam/to"
	"gopkg.in/yaml.v2"

	"github.com/moira-alert/moira/database/redis"
)

// RedisConfig is a redis config structure that initialises at the start of moira.
// Redis configuration depends on fields specified in redis config section:
// 1. Use fields MasterName and Addrs to enable Redis Sentinel support.
// 2. Specify two or more comma-separated Addrs to enable cluster support.
// 3. Otherwise, standalone configuration is enabled.
type RedisConfig struct {
	// Redis Sentinel master name
	MasterName string `yaml:"master_name"`
	// Redis address list, format: {host1_name:port},{ip:port}
	Addrs string `yaml:"addrs"`
	// Redis Sentinel password
	SentinelPassword string `yaml:"sentinel_password"`
	// Redis Sentinel username
	SentinelUsername string `yaml:"sentinel_username"`
	// Redis username
	Username string `yaml:"username"`
	// Redis password
	Password string `yaml:"password"`
	// Moira will delete metrics older than this value from Redis. Large values will lead to various problems everywhere.
	// See https://github.com/moira-alert/moira/pull/519
	MetricsTTL string `yaml:"metrics_ttl"`
	// Dial connection timeout. Default is 500ms.
	DialTimeout string `yaml:"dial_timeout"`
	// Read-operation timeout. Default is 3000ms.
	ReadTimeout string `yaml:"read_timeout"`
	// Write-operation timeout. Default is ReadTimeout seconds.
	WriteTimeout string `yaml:"write_timeout"`
	// MaxRetries count of retries.
	MaxRetries int `yaml:"max_retries"`
	// Enables read-only commands on slave nodes.
	ReadOnly bool `yaml:"read_only"`
	// Allows routing read-only commands to the **closest** master or slave node.
	// It automatically enables ReadOnly.
	RouteByLatency bool `yaml:"route_by_latency"`
	// Allows routing read-only commands to the **random** master or slave node.
	// It automatically enables ReadOnly.
	RouteRandomly bool `yaml:"route_randomly"`
}

// GetSettings returns redis config parsed from moira config files.
func (config *RedisConfig) GetSettings() redis.DatabaseConfig {
	return redis.DatabaseConfig{
		MasterName:     config.MasterName,
		Addrs:          strings.Split(config.Addrs, ","),
		Username:       config.Username,
		Password:       config.Password,
		MaxRetries:     config.MaxRetries,
		MetricsTTL:     to.Duration(config.MetricsTTL),
		DialTimeout:    to.Duration(config.DialTimeout),
		ReadTimeout:    to.Duration(config.ReadTimeout),
		WriteTimeout:   to.Duration(config.WriteTimeout),
		ReadOnly:       config.ReadOnly,
		RouteByLatency: config.RouteByLatency,
		RouteRandomly:  config.RouteRandomly,
	}
}

// NotificationHistoryConfig is the config which coordinates interaction with notification statistics.
// E.g. how much time should we store it, or how many history items can we request from database.
type NotificationHistoryConfig struct {
	// Time which moira should store contacts and theirs events history
	NotificationHistoryTTL string `yaml:"ttl"`
	// Max count of events which moira may send as response of contact and its events history
	NotificationHistoryQueryLimit int `yaml:"query_limit"`
}

// GetSettings returns notification history storage policy configuration.
func (notificationHistoryConfig *NotificationHistoryConfig) GetSettings() redis.NotificationHistoryConfig {
	return redis.NotificationHistoryConfig{
		NotificationHistoryTTL:        to.Duration(notificationHistoryConfig.NotificationHistoryTTL),
		NotificationHistoryQueryLimit: notificationHistoryConfig.NotificationHistoryQueryLimit,
	}
}

// NotificationConfig is a config that stores the necessary configuration of the notifier.
type NotificationConfig struct {
	// Need to determine if notification is delayed - the difference between creation time and sending time
	// is greater than DelayedTime
	DelayedTime string `yaml:"delayed_time"`
	// TransactionTimeout defines the timeout between fetch notifications transactions
	TransactionTimeout string `yaml:"transaction_timeout"`
	// TransactionMaxRetries defines the maximum number of attempts to make a transaction
	TransactionMaxRetries int `yaml:"transaction_max_retries"`
	// TransactionHeuristicLimit maximum allowable limit, after this limit all notifications
	// without limit will be taken
	TransactionHeuristicLimit int64 `yaml:"transaction_heuristic_limit"`
	// ResaveTime is the time by which the timestamp of notifications with triggers
	// or metrics on Maintenance is incremented
	ResaveTime string `yaml:"resave_time"`
}

// GetSettings returns notification storage configuration.
func (notificationConfig *NotificationConfig) GetSettings() redis.NotificationConfig {
	return redis.NotificationConfig{
		DelayedTime:               to.Duration(notificationConfig.DelayedTime),
		TransactionTimeout:        to.Duration(notificationConfig.TransactionTimeout),
		TransactionMaxRetries:     notificationConfig.TransactionMaxRetries,
		TransactionHeuristicLimit: notificationConfig.TransactionHeuristicLimit,
		ResaveTime:                to.Duration(notificationConfig.ResaveTime),
	}
}

// GraphiteConfig is graphite metrics config structure that initialises at the start of moira.
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

// GetSettings returns graphite metrics config parsed from moira config files.
func (graphiteConfig *GraphiteConfig) GetSettings() metrics.GraphiteRegistryConfig {
	return metrics.GraphiteRegistryConfig{
		Enabled:      graphiteConfig.Enabled,
		RuntimeStats: graphiteConfig.RuntimeStats,
		URI:          graphiteConfig.URI,
		Prefix:       graphiteConfig.Prefix,
		Interval:     to.Duration(graphiteConfig.Interval),
	}
}

// LoggerConfig is logger settings structure that initialises at the start of moira.
type LoggerConfig struct {
	LogFile         string `yaml:"log_file"`
	LogLevel        string `yaml:"log_level"`
	LogPrettyFormat bool   `yaml:"log_pretty_format"`
}

// TelemetryConfig is settings for listener, pprof, graphite.
type TelemetryConfig struct {
	Listen   string         `yaml:"listen"`
	Pprof    ProfilerConfig `yaml:"pprof"`
	Graphite GraphiteConfig `yaml:"graphite"`
}

// ProfilerConfig is pprof settings structure that initialises at the start of moira.
type ProfilerConfig struct {
	Enabled bool `yaml:"enabled"`
}

// RemotesConfig is designed to be embedded in config files to configure all remote sources.
type RemotesConfig struct {
	Graphite   []GraphiteRemoteConfig   `yaml:"graphite_remote"`
	Prometheus []PrometheusRemoteConfig `yaml:"prometheus_remote"`
}

// Validate returns nil if config is valid, or error if it is malformed.
func (remotes *RemotesConfig) Validate() error {
	errs := make([]error, 0)

	errs = append(errs, validateRemotes[GraphiteRemoteConfig](remotes.Graphite)...)
	errs = append(errs, validateRemotes[PrometheusRemoteConfig](remotes.Prometheus)...)

	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func validateRemotes[T remoteCommon](remotes []T) []error {
	errs := make([]error, 0)

	keys := make(map[moira.ClusterId]int)
	for _, remote := range remotes {
		common := remote.getRemoteCommon()
		if common.ClusterId == moira.ClusterNotSet {
			err := fmt.Errorf("cluster id must be set for remote source (name: `%s`, url: `%s`)",
				common.ClusterName, common.URL,
			)
			errs = append(errs, err)
		}
		keys[common.ClusterId]++
	}

	for key, count := range keys {
		if count > 1 {
			err := fmt.Errorf("cluster id must be unique, non unique cluster id found: %s", key.String())
			errs = append(errs, err)
		}
	}

	return errs
}

// RemoteCommonConfig is designed to be embedded in remote configs, It contains fields that are similar for all remotes.
type RemoteCommonConfig struct {
	// Unique id of the cluster
	ClusterId moira.ClusterId `yaml:"cluster_id"`
	// Human-readable name of the cluster
	ClusterName string `yaml:"cluster_name"`
	// graphite url e.g http://graphite/render
	URL string `yaml:"url"`
	// Min period to perform triggers re-check. Note: Reducing of this value leads to increasing of CPU and memory usage values
	CheckInterval string `yaml:"check_interval"`
	// Number of checks that can be run in parallel
	// If empty will be set to number of cpu cores
	MaxParallelChecks int `yaml:"max_parallel_checks"`
	// Moira won't fetch metrics older than this value from remote storage. Note that Moira doesn't delete old data from
	// remote storage. Large values will lead to OOM problems in checker.
	// See https://github.com/moira-alert/moira/pull/519
	MetricsTTL string `yaml:"metrics_ttl"`
}

type remoteCommon interface {
	getRemoteCommon() *RemoteCommonConfig
}

// GraphiteRemoteConfig is remote graphite settings structure.
type GraphiteRemoteConfig struct {
	RemoteCommonConfig `yaml:",inline"`
	// Timeout for remote requests
	Timeout string `yaml:"timeout"`
	// Username for basic auth
	User string `yaml:"user"`
	// Password for basic auth
	Password string `yaml:"password"`
}

func (config GraphiteRemoteConfig) getRemoteCommon() *RemoteCommonConfig {
	return &config.RemoteCommonConfig
}

// GetRemoteSourceSettings returns remote config parsed from moira config files.
func (config *GraphiteRemoteConfig) GetRemoteSourceSettings() *graphiteRemoteSource.Config {
	return &graphiteRemoteSource.Config{
		URL:           config.URL,
		CheckInterval: to.Duration(config.CheckInterval),
		MetricsTTL:    to.Duration(config.MetricsTTL),
		Timeout:       to.Duration(config.Timeout),
		User:          config.User,
		Password:      config.Password,
	}
}

// GraphiteRemoteConfig is remote prometheus settings structure.
type PrometheusRemoteConfig struct {
	RemoteCommonConfig `yaml:",inline"`
	// Timeout for prometheus api requests
	Timeout string `yaml:"timeout"`
	// Number of retries for prometheus api requests
	Retries int `yaml:"retries"`
	// Timeout between retries for prometheus api requests
	RetryTimeout string `yaml:"retry_timeout"`
	// Username for basic auth
	User string `yaml:"user"`
	// Password for basic auth
	Password string `yaml:"password"`
}

func (config PrometheusRemoteConfig) getRemoteCommon() *RemoteCommonConfig {
	return &config.RemoteCommonConfig
}

// GetRemoteSourceSettings returns remote config parsed from moira config files.
func (config *PrometheusRemoteConfig) GetPrometheusSourceSettings() *prometheusRemoteSource.Config {
	return &prometheusRemoteSource.Config{
		URL:            config.URL,
		CheckInterval:  to.Duration(config.CheckInterval),
		MetricsTTL:     to.Duration(config.MetricsTTL),
		User:           config.User,
		Password:       config.Password,
		RequestTimeout: to.Duration(config.Timeout),
		Retries:        config.Retries,
		RetryTimeout:   to.Duration(config.RetryTimeout),
	}
}

// ImageStoreConfig defines the configuration for all the image stores to be initialized by InitImageStores.
type ImageStoreConfig struct {
	S3 s3.Config `yaml:"s3"`
}

// ReadConfig parses config file by the given path into Moira-used type.
func ReadConfig(configFileName string, config interface{}) error {
	configYaml, err := os.ReadFile(configFileName)
	if err != nil {
		return fmt.Errorf("can't read file [%s] [%s]", configFileName, err.Error())
	}
	err = yaml.Unmarshal(configYaml, config)
	if err != nil {
		return fmt.Errorf("can't parse config file [%s] [%s]", configFileName, err.Error())
	}
	return nil
}

// PrintConfig prints config to stdout.
func PrintConfig(config interface{}) {
	d, _ := yaml.Marshal(&config)
	fmt.Println(string(d))
}
