package notifier

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/wcharczuk/go-chart"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/plotting"
)

var (
	// defaultTimeShift is default time shift to fetch timeseries
	defaultTimeShift = 1 * time.Minute
	// defaultTimeRange is default time range to fetch timeseries
	defaultTimeRange = 30 * time.Minute
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

// buildNotificationPackagePlot returns bytes slice containing package plot
func (notifier *StandardNotifier) buildNotificationPackagePlot(pkg NotificationPackage) ([]byte, error) {
	buff := bytes.NewBuffer(make([]byte, 0))
	if !pkg.Plotting.Enabled {
		return buff.Bytes(), nil
	}
	if pkg.Trigger.ID == "" {
		return buff.Bytes(), nil
	}
	metricsToShow := pkg.GetMetricNames()
	if len(metricsToShow) == 0 {
		return buff.Bytes(), nil
	}
	plotTemplate, err := plotting.GetPlotTemplate(pkg.Plotting.Theme, notifier.config.Location)
	if err != nil {
		return buff.Bytes(), err
	}

	from, to := resolveMetricsWindow(notifier.logger, pkg.Trigger, pkg)
	metricsData, trigger, err := notifier.evaluateTriggerMetrics(from, to, pkg.Trigger.ID)
	if err != nil {
		return buff.Bytes(), err
	}
	notifier.logger.Debugf("rendering %s timeseries: %s", trigger.ID, strings.Join(metricsToShow, ", "))
	var md = make([]*metricSource.MetricData, 0, len(metricsData))
	for _, metricData := range metricsData {
		md = append(md, &metricSource.MetricData{
			Name:      metricData.Name,
			StartTime: metricData.StartTime,
			StopTime:  metricData.StopTime,
			StepTime:  metricData.StepTime,
			Values:    metricData.Values,
		})
	}
	renderable, err := plotTemplate.GetRenderable(trigger, md, metricsToShow)
	if err != nil {
		return buff.Bytes(), err
	}
	if err = renderable.Render(chart.PNG, buff); err != nil {
		return buff.Bytes(), err
	}
	return buff.Bytes(), nil
}

// resolveMetricsWindow returns from, to parameters depending on trigger type
func resolveMetricsWindow(logger moira.Logger, trigger moira.TriggerData, pkg NotificationPackage) (int64, int64) {
	// resolve default realtime window for any case
	now := time.Now()
	defaultFrom := now.UTC().Add(-defaultTimeRange).Unix()
	defaultTo := now.UTC().Unix()
	// try to resolve package window, force default realtime window on fail for both local and remote triggers
	from, to, err := pkg.GetWindow()
	if err != nil {
		logger.Warningf("failed to get trigger %s package window: %s, using default %s window",
			trigger.ID, err.Error(), defaultTimeRange.String())
		return defaultFrom, defaultTo
	}
	// package window successfully resolved, test it's wide and realtime metrics window
	fromTime, toTime := moira.Int64ToTime(from), moira.Int64ToTime(to)
	isWideWindow := toTime.Sub(fromTime).Minutes() >= defaultTimeRange.Minutes()
	isRealTimeWindow := now.UTC().Sub(fromTime).Minutes() <= defaultTimeRange.Minutes()
	// resolve remote trigger window
	// window is wide: use package window to fetch limited historical data from graphite
	// window is not wide: use shifted window to fetch extended historical data from graphite
	if trigger.IsRemote {
		if isWideWindow {
			return fromTime.Unix(), toTime.Unix()
		}
		return toTime.Add(-defaultTimeRange + defaultTimeShift).Unix(), toTime.Add(defaultTimeShift).Unix()
	}
	// resolve local trigger window
	// window is realtime: use shifted window to fetch actual data from redis
	// window is not realtime: force realtime window
	if isRealTimeWindow {
		return toTime.Add(-defaultTimeRange + defaultTimeShift).Unix(), toTime.Add(defaultTimeShift).Unix()
	}
	return defaultFrom, defaultTo
}

// evaluateTriggerMetrics returns collection of MetricData
func (notifier *StandardNotifier) evaluateTriggerMetrics(from, to int64, triggerID string) ([]*metricSource.MetricData, *moira.Trigger, error) {
	trigger, err := notifier.database.GetTrigger(triggerID)
	if err != nil {
		return nil, nil, err
	}
	metricsSource, err := notifier.metricSourceProvider.GetTriggerMetricSource(&trigger)
	if err != nil {
		return nil, &trigger, err
	}
	var metricsData = make([]*metricSource.MetricData, 0)
	for _, target := range trigger.Targets {
		timeSeries, fetchErr := fetchAvailableSeries(metricsSource, target, from, to)
		if fetchErr != nil {
			return nil, &trigger, fetchErr
		}
		metricsData = append(metricsData, timeSeries...)
	}
	return metricsData, &trigger, err
}

// fetchAvailableSeries calls fetch function with realtime alerting and retries on fail without
func fetchAvailableSeries(metricsSource metricSource.MetricSource, target string, from, to int64) ([]*metricSource.MetricData, error) {
	realtimeFetchResult, realtimeErr := metricsSource.Fetch(target, from, to, true)
	switch realtimeErr.(type) {
	case local.ErrEvaluateTargetFailedWithPanic:
		fetchResult, err := metricsSource.Fetch(target, from, to, false)
		if err != nil {
			return nil, errFetchAvailableSeriesFailed{realtimeErr: realtimeErr.Error(), storedErr: err.Error()}
		}
		return fetchResult.GetMetricsData(), nil
	}
	return realtimeFetchResult.GetMetricsData(), realtimeErr
}
