package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"

	prometheusApi "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const StepTimeSeconds int64 = 60

type Config struct {
	Enabled       bool
	CheckInterval time.Duration
	MetricsTTL    time.Duration
	Timeout       time.Duration
	URL           string
	User          string
	Password      string
}

func Create(config *Config) (metricSource.MetricSource, error) {
	promApi, err := createPrometheusApi(config)

	if err != nil {
		return nil, err
	}

	return &Prometheus{
		config: config,
		api:    promApi,
	}, nil
}

type Prometheus struct {
	config *Config
	api    prometheusApi.API
}

func (prometheus *Prometheus) Fetch(target string, from int64, until int64, allowRealTimeAlerting bool) (metricSource.FetchResult, error) {
	from = moira.MaxInt64(from, until-int64(prometheus.config.MetricsTTL.Seconds()))

	ctx, cancel := context.WithTimeout(context.Background(), prometheus.config.Timeout)
	defer cancel()

	val, _, err := prometheus.api.QueryRange(ctx, target, prometheusApi.Range{
		Start: time.Unix(from, 0),
		End:   time.Unix(until, 0),
		Step:  time.Second * time.Duration(StepTimeSeconds),
	})
	if err != nil {
		return nil, err
	}

	mat := val.(model.Matrix)

	return convertToFetchResult(mat), nil
}

func (prometheus *Prometheus) GetMetricsTTLSeconds() int64 {
	return int64(prometheus.config.MetricsTTL.Seconds())
}

func (prometheus *Prometheus) IsConfigured() (bool, error) {
	// TODO: check if configuration is valid
	return prometheus.config.Enabled, nil
}

func (*Prometheus) IsAvailable() (bool, error) {
	// TODO: check if prometheus is actually available
	return true, nil
}

type FetchResult struct {
	MetricsData []metricSource.MetricData
}

// GetMetricsData return all metrics data from fetch result
func (fetchResult *FetchResult) GetMetricsData() []metricSource.MetricData {
	return fetchResult.MetricsData
}

// GetPatterns always returns error, because we can't fetch target patterns from remote metrics source
func (*FetchResult) GetPatterns() ([]string, error) {
	return make([]string, 0), fmt.Errorf("remote fetch result never returns patterns")
}

// GetPatternMetrics always returns error, because remote fetch doesn't return base pattern metrics
func (*FetchResult) GetPatternMetrics() ([]string, error) {
	return make([]string, 0), fmt.Errorf("remote fetch result never returns pattern metrics")
}
