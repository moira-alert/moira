package notifier

import (
	"bytes"
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

// defaultTimeRange is default time range to fetch timeseries
var defaultTimeRange = 60 * time.Minute

// buildNotificationPackagePlot returns bytes slice containing package plot
func (notifier *StandardNotifier) buildNotificationPackagePlot(pkg NotificationPackage) ([]byte, error) {
	buff := bytes.NewBuffer(make([]byte, 0))
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
	metricsToShow := pkg.MetricNames()
	renderable := plotTemplate.GetRenderable(trigger, metricsData, metricsToShow)
	notifier.logger.Debugf("processing render %s timeseries: %s with location: %s",
		trigger.ID, strings.Join(metricsToShow, ", "), notifier.config.Location.String())
	if len(renderable.Series) == 0 {
		return buff.Bytes(), plotting.ErrNoPointsToRender{TriggerName: trigger.Name}
	}
	if err = renderable.Render(chart.PNG, buff); err != nil {
		return buff.Bytes(), err
	}
	return buff.Bytes(), nil
}

// resolveMetricsWindow returns from, to parameters depending on trigger type
func resolveMetricsWindow(logger moira.Logger, trigger moira.TriggerData, pkg NotificationPackage) (int64, int64) {
	var err error
	var from, to int64
	if trigger.IsRemote {
		from, to, err = pkg.Window()
	}
	if !trigger.IsRemote || err != nil {
		if err != nil {
			logger.Warningf("can not use remote trigger %s window: %s, using default %s window",
				trigger.ID, err.Error(), defaultTimeRange.String())
		}
		now := time.Now()
		from = now.UTC().Add(-defaultTimeRange).Unix()
		to = now.UTC().Unix()
	}
	return from, to
}

// evaluateTriggerMetrics returns collection of MetricData
func evaluateTriggerMetrics(database moira.Database, remoteCfg *remote.Config,
	from, to int64, triggerID string) ([]*types.MetricData, *moira.Trigger, error) {
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
func getTriggerEvaluationResult(dataBase moira.Database, remoteConfig *remote.Config,
	from, to int64, triggerID string) (*checker.TriggerTimeSeries, *moira.Trigger, error) {
	allowRealtimeAlerting := true
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
		var timeSeries []*target.TimeSeries
		if trigger.IsRemote {
			timeSeries, err = remote.Fetch(remoteConfig, tar, from, to, allowRealtimeAlerting)
			if err != nil {
				return nil, &trigger, err
			}
		} else {
			result, err := target.EvaluateTarget(dataBase, tar, from, to, allowRealtimeAlerting)
			if err != nil {
				return nil, &trigger, err
			}
			timeSeries = result.TimeSeries
		}
		if i == 0 {
			triggerMetrics.Main = timeSeries
		} else {
			triggerMetrics.Additional = append(triggerMetrics.Additional, timeSeries...)
		}
	}
	return triggerMetrics, &trigger, nil
}

