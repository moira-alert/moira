package prometheus

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
)

const StepTimeSeconds int64 = 60

var ErrPrometheusStorageDisabled = fmt.Errorf("remote prometheus storage is not enabled")

type Config struct {
	CheckInterval  time.Duration
	MetricsTTL     time.Duration
	RequestTimeout time.Duration
	Retries        int
	RetryTimeout   time.Duration
	URL            string
	User           string
	Password       string
}

func Create(config *Config, logger moira.Logger) (metricSource.MetricSource, error) {
	promApi, err := createPrometheusApi(config)
	if err != nil {
		return nil, err
	}

	return &Prometheus{config: config, api: promApi, logger: logger}, nil
}

type Prometheus struct {
	config *Config
	logger moira.Logger
	api    PrometheusApi
}

func (prometheus *Prometheus) GetMetricsTTLSeconds() int64 {
	return int64(prometheus.config.MetricsTTL.Seconds())
}

func (prometheus *Prometheus) IsConfigured() (bool, error) {
	return true, nil
}

func (prometheus *Prometheus) IsAvailable() (bool, error) {
	now := time.Now().Unix()
	_, err := prometheus.Fetch("1", now, now, true)
	return err == nil, err
}
