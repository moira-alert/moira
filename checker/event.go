package checker

import "github.com/moira-alert/moira-alert"

func (triggerChecker *TriggerChecker) compareStates(metric string, currentState moira.MetricState, lastState moira.MetricState) (moira.MetricState, error) {
	return moira.MetricState{}, nil
}
