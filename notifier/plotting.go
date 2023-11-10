package notifier

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/moira-alert/go-chart"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/plotting"
)

const (
	// defaultTimeShift is default time shift to fetch timeseries
	defaultTimeShift = 1 * time.Minute
	// defaultTimeRange is default time range to fetch timeseries
	defaultTimeRange = 30 * time.Minute
	// defaultRetentionSeconds is the most common metric retention
	defaultRetentionSeconds = 60
)

// errFetchAvailableSeriesFailed is used in cases when fetchAvailableSeries failed after retry
type errFetchAvailableSeriesFailed struct {
	realtimeErr string
	storedErr   string
}

// Error is implementation of golang error interface for errFetchAvailableSeriesFailed struct
func (err errFetchAvailableSeriesFailed) Error() string {
	return fmt.Sprintf("Failed to fetch both realtime and stored data: [realtime]: %s, [stored]: %s", err.realtimeErr, err.storedErr)
}

// buildTriggerPlots returns bytes slices containing trigger plots
func buildTriggerPlots(trigger *moira.Trigger, metricsData map[string][]metricSource.MetricData,
	plotTemplate *plotting.Plot) ([][]byte, error) {
	result := make([][]byte, 0)
	for targetName, metrics := range metricsData {
		renderable, err := plotTemplate.GetRenderable(targetName, trigger, metrics)
		if err != nil {
			return nil, err
		}
		buff := bytes.NewBuffer(make([]byte, 0))
		if err = renderable.Render(chart.PNG, buff); err != nil {
			return nil, err
		}
		result = append(result, buff.Bytes())
	}
	return result, nil
}

// buildNotificationPackagePlots returns bytes slices containing package plots
func (notifier *StandardNotifier) buildNotificationPackagePlots(pkg NotificationPackage, logger moira.Logger) ([][]byte, error) {
	if !pkg.Plotting.Enabled {
		return nil, nil
	}
	if pkg.Trigger.ID == "" {
		return nil, nil
	}

	logger.Info().Msg("Start build plots for package")
	startTime := time.Now()
	metricsToShow := pkg.GetMetricNames()
	if len(metricsToShow) == 0 {
		return nil, nil
	}
	plotTemplate, err := plotting.GetPlotTemplate(pkg.Plotting.Theme, notifier.config.Location)
	if err != nil {
		return nil, err
	}

	from, to := resolveMetricsWindow(logger, pkg.Trigger, pkg)
	evaluateTriggerStartTime := time.Now()
	metricsData, trigger, err := notifier.evaluateTriggerMetrics(from, to, pkg.Trigger.ID)
	if err != nil {
		return nil, err
	}
	notifier.metrics.PlotsEvaluateTriggerDurationMs.Update(time.Since(evaluateTriggerStartTime).Milliseconds())

	metricsData = getMetricDataToShow(metricsData, metricsToShow)
	logger.Debug().
		Interface("metrics_data", metricsData).
		Msg("Build plot from MetricsData")

	buildPlotStartTime := time.Now()
	result, err := buildTriggerPlots(trigger, metricsData, plotTemplate)
	notifier.metrics.PlotsBuildDurationMs.Update(time.Since(buildPlotStartTime).Milliseconds())

	logger.Info().
		Int64("moira.plots.build_duration_ms", time.Since(startTime).Milliseconds()).
		Msg("Finished build plots for package")

	return result, err
}

// resolveMetricsWindow returns from, to parameters depending on trigger type
func resolveMetricsWindow(logger moira.Logger, trigger moira.TriggerData, pkg NotificationPackage) (int64, int64) {
	// resolve default realtime window for any case
	now := time.Now()
	defaultFrom := roundToRetention(now.UTC().Add(-defaultTimeRange).Unix())
	defaultTo := roundToRetention(now.UTC().Unix())
	// try to resolve package window, force default realtime window on fail for both local and remote triggers
	from, to, err := pkg.GetWindow()
	if err != nil {
		logger.Warning().
			String("default_window", defaultTimeRange.String()).
			Error(err).
			Msg("Failed to get trigger package window, using default window")
		return defaultFrom, defaultTo
	}
	// round to the nearest retention to correctly fetch data from redis
	from = roundToRetention(from)
	to = roundToRetention(to)
	// package window successfully resolved, test it's wide and realtime metrics window
	fromTime, toTime := moira.Int64ToTime(from), moira.Int64ToTime(to)
	isWideWindow := toTime.Sub(fromTime).Minutes() >= defaultTimeRange.Minutes()
	isRealTimeWindow := now.UTC().Sub(fromTime).Minutes() <= defaultTimeRange.Minutes()
	// resolve remote trigger window.
	// window is wide: use package window to fetch limited historical data from graphite
	// window is not wide: use shifted window to fetch extended historical data from graphite
	if trigger.GetTriggerSource() == moira.GraphiteRemote {
		if isWideWindow {
			return fromTime.Unix(), toTime.Unix()
		}
		return toTime.Add(-defaultTimeRange + defaultTimeShift).Unix(), toTime.Add(defaultTimeShift).Unix()
	}
	// resolve local trigger window.
	// window is realtime: use shifted window to fetch actual data from redis
	// window is not realtime: force realtime window
	if isRealTimeWindow {
		return toTime.Add(-defaultTimeRange + defaultTimeShift).Unix(), toTime.Add(defaultTimeShift).Unix()
	}
	return defaultFrom, defaultTo
}

func roundToRetention(unixTime int64) int64 {
	return moira.RoundToNearestRetention(unixTime, defaultRetentionSeconds)
}

// evaluateTriggerMetrics returns collection of MetricData
func (notifier *StandardNotifier) evaluateTriggerMetrics(from, to int64, triggerID string) (map[string][]metricSource.MetricData, *moira.Trigger, error) {
	trigger, err := notifier.database.GetTrigger(triggerID)
	if err != nil {
		return nil, nil, err
	}
	metricsSource, err := notifier.metricSourceProvider.GetTriggerMetricSource(&trigger)
	if err != nil {
		return nil, &trigger, err
	}
	var result = make(map[string][]metricSource.MetricData)
	for i, target := range trigger.Targets {
		i++ // Increase
		targetName := fmt.Sprintf("t%d", i)
		timeSeries, fetchErr := fetchAvailableSeries(metricsSource, target, from, to)
		if fetchErr != nil {
			return nil, &trigger, fetchErr
		}
		result[targetName] = timeSeries
	}
	return result, &trigger, err
}

// fetchAvailableSeries calls fetch function with realtime alerting and retries on fail without
func fetchAvailableSeries(metricsSource metricSource.MetricSource, target string, from, to int64) ([]metricSource.MetricData, error) {
	realtimeFetchResult, realtimeErr := metricsSource.Fetch(target, from, to, true)
	if realtimeErr == nil {
		return realtimeFetchResult.GetMetricsData(), nil
	}
	var errFailedWithPanic local.ErrEvaluateTargetFailedWithPanic
	if ok := errors.As(realtimeErr, &errFailedWithPanic); ok {
		fetchResult, err := metricsSource.Fetch(target, from, to, false)
		if err != nil {
			return nil, errFetchAvailableSeriesFailed{realtimeErr: errFailedWithPanic.Error(), storedErr: err.Error()}
		}
		return fetchResult.GetMetricsData(), nil
	}
	return nil, realtimeErr
}

// getMetricDataToShow returns MetricData limited by whitelist
func getMetricDataToShow(metricsData map[string][]metricSource.MetricData, metricsWhitelist []string) map[string][]metricSource.MetricData {
	result := make(map[string][]metricSource.MetricData)
	if len(metricsWhitelist) == 0 {
		return metricsData
	}
	metricsWhitelistHash := make(map[string]bool, len(metricsWhitelist))
	for _, whiteListed := range metricsWhitelist {
		metricsWhitelistHash[whiteListed] = true
	}

	for targetName, metrics := range metricsData {
		newMetricsData := make([]metricSource.MetricData, 0, len(metricsWhitelist))
		if len(metrics) == 1 {
			result[targetName] = metrics
			continue
		}
		for _, metricData := range metrics {
			if _, ok := metricsWhitelistHash[metricData.Name]; ok {
				newMetricsData = append(newMetricsData, metricData)
			}
		}
		result[targetName] = newMetricsData
	}
	return result
}
