package prometheus

import (
	"context"
	"fmt"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/prometheus/client_golang/api"
	apiV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"time"
)

// FetchResult is implementation of metric_source.FetchResult interface,
// which represent fetching result from remote graphite installation in moira format
type FetchResult struct {
	MetricsData []*metricSource.MetricData
}

// GetMetricsData return all metrics data from fetch result
func (fetchResult *FetchResult) GetMetricsData() []*metricSource.MetricData {
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

type Config struct {
	URL           string
	CheckInterval time.Duration
	Timeout       time.Duration
	User          string
	Password      string
	Enabled       bool
}

type Source struct {
	api    apiV1.API
	logger moira.Logger
}

func (source *Source) Fetch(target string, from int64, until int64, allowRealTimeAlerting bool) (metricSource.FetchResult, error) {
	source.logger.Debugf("Fetching metrics for target %s from %d until %d", target, from, until)
	value, _, err := source.api.QueryRange(
		context.TODO(),
		target,
		apiV1.Range{Start: time.Unix(from, 0), End: time.Unix(until, 0)},
	)
	if err != nil {
		return nil, err
	}
	metrics, ok := value.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("unsupported result format: %s", value.Type().String())
	}
	metricsData := make([]*metricSource.MetricData, 0, len(metrics))
	for _, metric := range metrics {

		metricValues := make([]float64, 0, len(metric.Values))
		for _, value := range metric.Values {
			metricValues = append(metricValues, float64(value.Value))
		}

		metricData := metricSource.MetricData{Values: metricValues}
		metricsData = append(metricsData, &metricData)
	}
	return &FetchResult{MetricsData: metricsData}, nil
}

func (source *Source) IsConfigured() (bool, error) {
	// TODO
	return true, nil
}

func (source *Source) IsAvailable() (bool, error) {
	// TODO
	return true, nil
}

// Create configures remote metric source
func Create(logger moira.Logger, config *Config) metricSource.MetricSource {
	client, _ := api.NewClient(api.Config{Address: config.URL})
	return &Source{api: apiV1.NewAPI(client), logger: logger}
}
