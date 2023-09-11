package prometheus

import (
	"context"
	"fmt"
	"time"

	metricSource "github.com/moira-alert/moira/metric_source"

	"github.com/moira-alert/moira"
	prometheusApi "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

func (prometheus *Prometheus) Fetch(target string, from, until int64, allowRealTimeAlerting bool) (metricSource.FetchResult, error) {
	from = moira.MaxInt64(from, until-int64(prometheus.config.MetricsTTL.Seconds()))

	var err error
	for i := 0; i < prometheus.config.Retries; i++ {
		var res metricSource.FetchResult
		res, err = prometheus.fetch(target, from, until, allowRealTimeAlerting)

		if err == nil {
			return res, nil
		}

		prometheus.logger.Warning().
			Error(err).
			Int("retries left", prometheus.config.Retries-i-1).
			Msg("Failed to fetch prometheus target")

		time.Sleep(prometheus.config.RetryTimeout)
	}

	return nil, err
}

func (prometheus *Prometheus) fetch(target string, from, until int64, allowRealTimeAlerting bool) (metricSource.FetchResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), prometheus.config.RequestTimeout)
	defer cancel()

	val, warns, err := prometheus.api.QueryRange(ctx, target, prometheusApi.Range{
		Start: time.Unix(from, 0),
		End:   time.Unix(until, 0),
		Step:  time.Second * time.Duration(StepTimeSeconds),
	})

	if len(warns) != 0 {
		prometheus.logger.
			Warning().
			Interface("warns", warns).
			Msg("Warnings when fetching metrics from remote prometheus")
	}

	if err != nil {
		return nil, err
	}

	mat := val.(model.Matrix)

	return convertToFetchResult(mat, from, until, allowRealTimeAlerting), nil
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
