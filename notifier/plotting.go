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

func (notifier *StandardNotifier) buildNotificationPackagePlot(pkg NotificationPackage,
	location *time.Location) ([]byte, error) {

	buff := bytes.NewBuffer(make([]byte, 0))

	if pkg.Trigger.ID == "" {
		return buff.Bytes(), nil
	}

	trigger, err := notifier.database.GetTrigger(pkg.Trigger.ID)
	if err != nil {
		return buff.Bytes(), err
	}

	plotTemplate, err := plotting.GetPlotTemplate(pkg.Plotting.Theme, location)
	if err != nil {
		return buff.Bytes(), err
	}

	remoteCfg := notifier.config.RemoteConfig

	to := time.Now().UTC()
	from := to.Add(-60 * time.Minute)

	tts, err := getTriggerEvaluationResult(notifier.database, remoteCfg, from.Unix(), to.Unix(), trigger.ID)
	if err != nil {
		return buff.Bytes(), err
	}

	var metricsData = make([]*types.MetricData, 0, len(tts.Main)+len(tts.Additional))
	for _, ts := range tts.Main {
		metricsData = append(metricsData, &ts.MetricData)
	}
	for _, ts := range tts.Additional {
		metricsData = append(metricsData, &ts.MetricData)
	}

	metricsToShow := make([]string, 0)

	for _, event := range pkg.Events {
		metricsToShow = append(metricsToShow, event.Metric)
	}

	renderable := plotTemplate.GetRenderable(&trigger, metricsData, metricsToShow)

	notifier.logger.Debugf("Attempt to render %s timeseries: %s", trigger.ID,
		strings.Join(metricsToShow, ", "))

	if err = renderable.Render(chart.PNG, buff); err != nil {
		return buff.Bytes(), err
	}

	return buff.Bytes(), nil
}

func getTriggerEvaluationResult(dataBase moira.Database, remoteConfig *remote.Config,
	from, to int64, triggerID string) (*checker.TriggerTimeSeries, error) {
	allowRealtimeAllerting := true
	trigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		return nil, err
	}
	triggerMetrics := &checker.TriggerTimeSeries{
		Main:       make([]*target.TimeSeries, 0),
		Additional: make([]*target.TimeSeries, 0),
	}
	if trigger.IsRemote && !remoteConfig.IsEnabled() {
		return nil, remote.ErrRemoteStorageDisabled
	}
	for i, tar := range trigger.Targets {
		var timeSeries []*target.TimeSeries
		if trigger.IsRemote {
			timeSeries, err = remote.Fetch(remoteConfig, tar, from, to, allowRealtimeAllerting)
			if err != nil {
				return nil, err
			}
		} else {
			result, err := target.EvaluateTarget(dataBase, tar, from, to, allowRealtimeAllerting)
			if err != nil {
				return nil, err
			}
			timeSeries = result.TimeSeries
		}
		if i == 0 {
			triggerMetrics.Main = timeSeries
		} else {
			triggerMetrics.Additional = append(triggerMetrics.Additional, timeSeries...)
		}
	}
	return triggerMetrics, nil
}

