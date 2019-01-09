package notifier

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/plotting"
	"github.com/moira-alert/moira/remote"
	"github.com/moira-alert/moira/target"
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
	plotTemplate, err := plotting.GetPlotTemplate(pkg.Plotting.Theme, notifier.config.Location)
	if err != nil {
		return buff.Bytes(), err
	}
	remoteCfg := notifier.config.RemoteConfig
	from, to := resolveMetricsWindow(notifier.logger, pkg.Trigger, pkg)
	metricsData, trigger, err := evaluateTriggerMetrics(notifier.database, remoteCfg, from, to, pkg.Trigger.ID)
	if err != nil {
		return buff.Bytes(), err
	}
	metricsToShow := pkg.GetMetricNames()
	notifier.logger.Debugf("rendering %s timeseries: %s", trigger.ID, strings.Join(metricsToShow, ", "))
	renderable, err := plotTemplate.GetRenderable(trigger, metricsData, metricsToShow)
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
func evaluateTriggerMetrics(database moira.Database, remoteCfg *remote.Config, from, to int64, triggerID string) ([]*types.MetricData, *moira.Trigger, error) {
	tts, trigger, err := getTriggerEvaluationResult(database, remoteCfg, from, to, triggerID)
	if err != nil {
		return nil, trigger, err
	}
	var metricsData = make([]*types.MetricData, 0, len(tts.Main)+len(tts.Additional))
	for _, ts := range tts.Main {
		metricsData = append(metricsData, &ts.MetricData)
	}
	for _, ts := range tts.Additional {
		metricsData = append(metricsData, &ts.MetricData)
	}
	return metricsData, trigger, err
}

// getTriggerEvaluationResult returns trigger metrics from chosen data source
func getTriggerEvaluationResult(dataBase moira.Database, remoteConfig *remote.Config, from, to int64, triggerID string) (*checker.TriggerTimeSeries, *moira.Trigger, error) {
	trigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		return nil, nil, err
	}
	triggerMetrics := &checker.TriggerTimeSeries{
		Main:       make([]*target.TimeSeries, 0),
		Additional: make([]*target.TimeSeries, 0),
	}
	if trigger.IsRemote && !remoteConfig.IsEnabled() {
		return nil, &trigger, remote.ErrRemoteStorageDisabled
	}
	for i, tar := range trigger.Targets {
		timeSeries, err := fetchAvailableSeries(dataBase, remoteConfig, trigger.IsRemote, tar, from, to)
		if err != nil {
			return nil, &trigger, err
		}
		if i == 0 {
			triggerMetrics.Main = timeSeries
		} else {
			triggerMetrics.Additional = append(triggerMetrics.Additional, timeSeries...)
		}
	}
	return triggerMetrics, &trigger, nil
}

// fetchAvailableSeries calls fetch function with realtime alerting and retries on fail without
func fetchAvailableSeries(database moira.Database, remoteCfg *remote.Config, isRemote bool, tar string, from, to int64) ([]*target.TimeSeries, error) {
	var err error
	if isRemote {
		return remote.Fetch(remoteCfg, tar, from, to, true)
	}
	result, realtimeErr := target.EvaluateTarget(database, tar, from, to, true)
	switch realtimeErr.(type) {
	case target.ErrEvaluateTargetFailedWithPanic:
		result, err = target.EvaluateTarget(database, tar, from, to, false)
		if err != nil {
			return nil, errFetchAvailableSeriesFailed{realtimeErr:realtimeErr.Error(), storedErr:err.Error()}
		}
		return result.TimeSeries, nil
	}
	return result.TimeSeries, realtimeErr
}
