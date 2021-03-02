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
