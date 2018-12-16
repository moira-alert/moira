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

var defaultPlotWindow = 60 * time.Minute

// buildNotificationPackagePlot returns bytes slice containing package plot
func (notifier *StandardNotifier) buildNotificationPackagePlot(pkg NotificationPackage) ([]byte, error) {
	buff := bytes.NewBuffer(make([]byte, 0))

	if pkg.Trigger.ID == "" {
		return buff.Bytes(), nil
	}

	trigger, err := notifier.database.GetTrigger(pkg.Trigger.ID)
	if err != nil {
		return buff.Bytes(), err
	}

	plotTemplate, err := plotting.GetPlotTemplate(pkg.Plotting.Theme, notifier.config.Location)
	if err != nil {
		return buff.Bytes(), err
	}

	remoteCfg := notifier.config.RemoteConfig

	from, to := notifier.getPlotWindow(trigger, pkg)
	metricsData, _, _ := notifier.evaluateTriggerMetrics(remoteCfg, from, to, trigger.ID)

	metricsToShow := make([]string, 0)

	for _, event := range pkg.Events {
		metricsToShow = append(metricsToShow, event.Metric)
	}

	notifier.logger.Info("expected series: %s", strings.Join(metricsToShow, ", "))
	metricsToShow = make([]string, 0)

	renderable := plotTemplate.GetRenderable(&trigger, metricsData, metricsToShow)

	allseries := make([]string, 0)
	for _, serie := range renderable.Series {
		allseries = append(allseries, serie.GetName())
	}
	notifier.logger.Infof("found series: %s", allseries)

	notifier.logger.Debugf("Attempt to render %s timeseries: %s", trigger.ID,
		strings.Join(metricsToShow, ", "))

	if err = renderable.Render(chart.PNG, buff); err != nil {
		return buff.Bytes(), err
	}

	return buff.Bytes(), nil
}

func (notifier *StandardNotifier) getPlotWindow(trigger moira.Trigger, pkg NotificationPackage) (int64, int64) {
	var err error
	var from, to int64
	if trigger.IsRemote {
		from, to, err = pkg.Window()
	}
	if !trigger.IsRemote || err != nil {
		if err != nil {
			notifier.logger.Warningf("can not use remote trigger %s window: %s, using default %s window",
				trigger.ID, err.Error(), defaultPlotWindow.String())
		}
		now := time.Now()
		from = now.In(notifier.config.Location).Add(-defaultPlotWindow).Unix()
		to = now.In(notifier.config.Location).Unix()
	}
	return from, to
}

func (notifier *StandardNotifier) evaluateTriggerMetrics(remoteCfg *remote.Config,
	from, to int64, triggerID string) ([]*types.MetricData, *moira.Trigger, error) {
	tts, trigger, err := notifier.getTriggerEvaluationResult(notifier.database, remoteCfg, from, to, triggerID)
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

func (notifier *StandardNotifier) getTriggerEvaluationResult(dataBase moira.Database, remoteConfig *remote.Config,
	from, to int64, triggerID string) (*checker.TriggerTimeSeries, *moira.Trigger, error) {
	allowRealtimeAllerting := true
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
			timeSeries, err = remote.Fetch(remoteConfig, tar, from, to, allowRealtimeAllerting)
			if err != nil {
				return nil, &trigger, err
			}
		} else {
			result, err := target.EvaluateTarget(dataBase, tar, from, to, allowRealtimeAllerting)
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

