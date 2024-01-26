package main

import (
	"runtime"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/cmd"
	"github.com/xiam/to"
)

type config struct {
	Redis     cmd.RedisConfig     `yaml:"redis"`
	Logger    cmd.LoggerConfig    `yaml:"log"`
	Checker   checkerConfig       `yaml:"checker"`
	Telemetry cmd.TelemetryConfig `yaml:"telemetry"`
	Local     localCheckConfig    `yaml:"local"`
	Remotes   cmd.RemotesConfig   `yaml:",inline"`
}

type triggerLogConfig struct {
	ID    string `yaml:"id"`
	Level string `yaml:"level"`
}

type triggersLogConfig struct {
	TriggersToLevel []triggerLogConfig `yaml:"triggers"`
}

type localCheckConfig struct {
	CheckInterval     string `yaml:"check_interval"`
	MaxParallelChecks int    `yaml:"max_parallel_checks"`
}

type checkerConfig struct {
	// Period for every trigger to perform forced check on
	NoDataCheckInterval string `yaml:"nodata_check_interval"`
	// Period for every trigger to cancel forced check (earlier than 'NoDataCheckInterval') if no metrics were received
	StopCheckingInterval string `yaml:"stop_checking_interval"`
	// Max period to perform lazy triggers re-check. Note: lazy triggers are triggers which has no subscription for it. Moira will check its state less frequently.
	// Delay for check lazy trigger is random between LazyTriggersCheckInterval/2 and LazyTriggersCheckInterval.
	LazyTriggersCheckInterval string `yaml:"lazy_triggers_check_interval"`
	// Specify log level by entities
	SetLogLevel triggersLogConfig `yaml:"set_log_level"`
	// Metric event pop operation batch size
	MetricEventPopBatchSize int `yaml:"metric_event_pop_batch_size"`
	// Metric event pop operation delay
	MetricEventPopDelay string `yaml:"metric_event_pop_delay"`
}

func handleParallelChecks(parallelChecks *int) bool {
	if parallelChecks != nil && *parallelChecks == 0 {
		*parallelChecks = runtime.NumCPU()
		return true
	}

	return false
}

func (config *config) getSettings(logger moira.Logger) *checker.Config {
	logTriggersToLevel := make(map[string]string)
	for _, v := range config.Checker.SetLogLevel.TriggersToLevel {
		logTriggersToLevel[v.ID] = v.Level
	}
	logger.Info().
		Int("number_of_triggers", len(logTriggersToLevel)).
		Msg("Found dynamic log rules in config for some triggers")

	sourceCheckConfigs := make(map[moira.ClusterKey]checker.SourceCheckConfig)

	localCheckConfig := checker.SourceCheckConfig{
		CheckInterval:     to.Duration(config.Local.CheckInterval),
		MaxParallelChecks: config.Local.MaxParallelChecks,
	}
	if handleParallelChecks(&localCheckConfig.MaxParallelChecks) {
		logger.Info().
			Int("number_of_cpu", localCheckConfig.MaxParallelChecks).
			String("trigger_source", moira.GraphiteLocal.String()).
			String("cluster_id", "default").
			Msg("MaxParallelChecks is not configured, set it to the number of CPU")
	}
	sourceCheckConfigs[moira.MakeClusterKey(moira.GraphiteLocal, moira.DefaultCluster)] = localCheckConfig

	for _, remote := range config.Remotes.Graphite {
		checkConfig := checker.SourceCheckConfig{
			CheckInterval:     to.Duration(remote.CheckInterval),
			MaxParallelChecks: remote.MaxParallelChecks,
		}
		if handleParallelChecks(&checkConfig.MaxParallelChecks) {
			logger.Info().
				Int("number_of_cpu", checkConfig.MaxParallelChecks).
				String("trigger_source", moira.GraphiteRemote.String()).
				String("cluster_id", remote.ClusterId.String()).
				Msg("MaxParallelChecks is not configured, set it to the number of CPU")
		}
		sourceCheckConfigs[moira.MakeClusterKey(moira.GraphiteRemote, remote.ClusterId)] = checkConfig
	}

	for _, remote := range config.Remotes.Prometheus {
		checkConfig := checker.SourceCheckConfig{
			CheckInterval:     to.Duration(remote.CheckInterval),
			MaxParallelChecks: remote.MaxParallelChecks,
		}
		if handleParallelChecks(&checkConfig.MaxParallelChecks) {
			logger.Info().
				Int("number_of_cpu", checkConfig.MaxParallelChecks).
				String("trigger_source", moira.PrometheusRemote.String()).
				String("cluster_id", remote.ClusterId.String()).
				Msg("MaxParallelChecks is not configured, set it to the number of CPU")
		}
		sourceCheckConfigs[moira.MakeClusterKey(moira.PrometheusRemote, remote.ClusterId)] = checkConfig
	}

	return &checker.Config{
		SourceCheckConfigs:          sourceCheckConfigs,
		LazyTriggersCheckInterval:   to.Duration(config.Checker.LazyTriggersCheckInterval),
		NoDataCheckInterval:         to.Duration(config.Checker.NoDataCheckInterval),
		StopCheckingIntervalSeconds: int64(to.Duration(config.Checker.StopCheckingInterval).Seconds()),
		LogTriggersToLevel:          logTriggersToLevel,
		MetricEventPopBatchSize:     int64(config.Checker.MetricEventPopBatchSize),
		MetricEventPopDelay:         to.Duration(config.Checker.MetricEventPopDelay),
	}
}

func getDefault() config {
	return config{
		Redis: cmd.RedisConfig{
			Addrs:       "localhost:6379",
			MetricsTTL:  "1h",
			DialTimeout: "500ms",
		},
		Logger: cmd.LoggerConfig{
			LogFile:         "stdout",
			LogLevel:        "info",
			LogPrettyFormat: false,
		},
		Checker: checkerConfig{
			NoDataCheckInterval: "60s",
			/// CheckInterval:             "5s",
			LazyTriggersCheckInterval: "10m",
			StopCheckingInterval:      "30s",
		},
		Telemetry: cmd.TelemetryConfig{
			Listen: ":8092",
			Graphite: cmd.GraphiteConfig{
				Enabled:      false,
				RuntimeStats: false,
				URI:          "localhost:2003",
				Prefix:       "DevOps.Moira",
				Interval:     "60s",
			},
			Pprof: cmd.ProfilerConfig{Enabled: false},
		},
		Local: localCheckConfig{
			CheckInterval: "60s",
		},
		Remotes: cmd.RemotesConfig{},
	}
}
