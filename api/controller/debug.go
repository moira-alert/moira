package controller

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/support"
)

func PullTrigger(database moira.Database, logger moira.Logger, triggerID string) (*moira.Trigger, *api.ErrorResponse) {
	trigger, err := support.HandlePullTrigger(logger, database, triggerID)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	return trigger, nil
}

func PullTriggerMetrics(database moira.Database, logger moira.Logger, triggerID string) ([]support.PatternMetrics, *api.ErrorResponse) {
	metrics, err := support.HandlePullTriggerMetrics(logger, database, triggerID)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	return metrics, nil
}

func PushTrigger(database moira.Database, logger moira.Logger, trigger *moira.Trigger) (err *api.ErrorResponse) {
	e := support.HandlePushTrigger(logger, database, trigger)
	if e != nil {
		return api.ErrorInternalServer(e)
	}
	return nil
}

func PushTriggerMetrics(database moira.Database, logger moira.Logger, triggerID string, metrics []support.PatternMetrics) (err *api.ErrorResponse) {
	e := support.HandlePushTriggerMetrics(logger, database, triggerID, metrics)
	if e != nil {
		return api.ErrorInternalServer(e)
	}
	return nil
}
