package checker

import (
	"fmt"
	"github.com/go-graphite/carbonapi/expr"
	"github.com/moira-alert/moira-alert"
	"time"
)

type TriggerChecker struct {
	TriggerId   string
	Database    moira.Database
	Logger      moira.Logger
	maintenance int64
	trigger     *moira.Trigger
	isSimple    bool
	ttl         *int64
	ttlState    string
	lastCheck   *moira.CheckData
}

type TimeSeries struct {
	*expr.MetricData
	lastState moira.MetricData
}

func (triggerChecker *TriggerChecker) Check(from *int64, now *int64) error {
	if now == nil {
		n := time.Now().Unix()
		now = &n
	}

	initialized, err := triggerChecker.initTrigger(from, *now)
	if err != nil || !initialized {
		return err
	}

	var fromTime int64
	if from == nil {
		fromTime = *triggerChecker.lastCheck.Timestamp
	}

	if triggerChecker.ttl != nil {
		fromTime = fromTime - *triggerChecker.ttl
	} else {
		fromTime = fromTime - 600
	}

	check := moira.CheckData{
		Metrics:   triggerChecker.lastCheck.Metrics,
		State:     "OK",
		Timestamp: now,
		Score:     triggerChecker.lastCheck.Score,
	}

	if err := triggerChecker.handleTrigger(check); err != nil {
		triggerChecker.Logger.Errorf("Trigger check failed: %s", err.Error())
		check.State = "EXCEPTION"
		check.Message = "Trigger evaluation exception"
		//todo compare_states
		return nil //is it right?
	}
	triggerChecker.Database.SetTriggerLastCheck(triggerChecker.TriggerId, &check)
	return nil
}

func (triggerChecker *TriggerChecker) handleTrigger(checkData moira.CheckData) error {

	return nil
}

func (triggerChecker *TriggerChecker) getTimeSeries(from, until int64) (*TargetTimeSeries, error) {
	targets := triggerChecker.trigger.Targets
	targetTimeSeries := TargetTimeSeries{
		OtherTargetsNames: make(map[string]string),
		TimeSeries:        make(map[int][]TimeSeries),
	}
	targetNumber := 1

	for _, target := range targets {
		metricDatas, err := EvaluateTarget(triggerChecker.Database, target, from, until, triggerChecker.isSimple)
		if err != nil {
			return nil, err
		}

		if targetNumber > 1 {
			if len(metricDatas) == 1 {
				targetTimeSeries.OtherTargetsNames[fmt.Sprintf("t%v", targetNumber)] = metricDatas[0].Name
			} else if len(metricDatas) == 0 {
				return nil, fmt.Errorf("Target #%s has no timeseries", targetNumber)
			} else {
				return nil, fmt.Errorf("Target #%s has more than one timeseries", targetNumber)
			}
		}

		timeSeriesArr := make([]TimeSeries, 0, len(metricDatas))
		for _, metric := range metricDatas {
			timeSeries := TimeSeries{
				MetricData: metric,
				lastState:  triggerChecker.getMetricLastCheck(metric),
			}
			timeSeriesArr = append(timeSeriesArr, timeSeries)
		}
		targetTimeSeries.TimeSeries[targetNumber] = timeSeriesArr
		targetNumber += 1
	}
	return &targetTimeSeries, nil
}

func (triggerChecker *TriggerChecker) getMetricLastCheck(data *expr.MetricData) moira.MetricData {
	metricData, ok := triggerChecker.lastCheck.Metrics[data.Name]
	if !ok {
		return moira.MetricData{
			State:     "NODATA",
			Timestamp: int64(data.StartTime - 3600),
		}
	} else {
		return metricData
	}
}

func (triggerChecker *TriggerChecker) initTrigger(fromTime *int64, now int64) (bool, error) {
	trigger, err := triggerChecker.Database.GetTrigger(triggerChecker.TriggerId)
	if err != nil {
		return false, err
	}
	if trigger == nil {
		return false, nil
	}

	triggerChecker.trigger = trigger
	triggerChecker.isSimple = trigger.IsSimpleTrigger

	tagDatas, err := triggerChecker.Database.GetTags(trigger.Tags)
	if err != nil {
		return false, err
	}

	for _, tagData := range tagDatas {
		if tagData.Maintenance != nil && *tagData.Maintenance > triggerChecker.maintenance {
			triggerChecker.maintenance = *tagData.Maintenance
			break
		}
	}

	triggerChecker.ttl = trigger.Ttl
	if trigger.TtlState != nil {
		triggerChecker.ttlState = *trigger.TtlState
	} else {
		triggerChecker.ttlState = "NODATA"
	}

	triggerChecker.lastCheck, err = triggerChecker.Database.GetTriggerLastCheck(triggerChecker.TriggerId)
	if err != nil {
		return false, err
	}

	var begin int64
	if fromTime != nil {
		begin = *fromTime - 3600
	} else {
		begin = now - 3600
	}
	if triggerChecker.lastCheck == nil {
		triggerChecker.lastCheck = &moira.CheckData{
			Metrics:   make(map[string]moira.MetricData),
			State:     "NODATA",
			Timestamp: &begin,
		}
	}

	if triggerChecker.lastCheck.Timestamp == nil {
		triggerChecker.lastCheck.Timestamp = &begin
	}

	return true, nil
}
