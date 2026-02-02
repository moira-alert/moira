package controller

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
)

// GetNotifierState return current notifier state.
func GetNotifierState(database moira.Database, clock moira.Clock) (*dto.NotifierState, *api.ErrorResponse) {
	state, err := database.GetNotifierState(clock)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	notifierState := dto.NotifierState{
		Actor: state.Actor,
		State: state.State,
	}
	if state.State == moira.SelfStateERROR {
		notifierState.Message = dto.ErrorMessage
	}

	return &notifierState, nil
}

// GetNotifierStatesForSources return current notifier state for all metric sources.
func GetNotifierStatesForSources(database moira.Database, clock moira.Clock) (*dto.NotifierStatesForSources, *api.ErrorResponse) {
	states, err := database.GetNotifierStateForSources(clock)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	notifierStates := dto.NotifierStatesForSources{
		Sources: []dto.NotifierStateForSource{},
	}

	for key, state := range states {
		notifierState := dto.NotifierStateForSource{
			NotifierState: dto.NotifierState{
				Actor: state.Actor,
				State: state.State,
			},
			TriggerSource: key.TriggerSource,
			ClusterId:     key.ClusterId,
		}
		if state.State == moira.SelfStateERROR {
			notifierState.Message = dto.ErrorMessageForSource(key)
		}

		notifierStates.Sources = append(notifierStates.Sources, notifierState)
	}

	return &notifierStates, nil
}

// UpdateNotifierState update current notifier state.
func UpdateNotifierState(database moira.Database, state *dto.NotifierState, clock moira.Clock) *api.ErrorResponse {
	err := database.SetNotifierState(moira.SelfStateActorManual, state.State, clock)
	if err != nil {
		return api.ErrorInternalServer(err)
	}

	return nil
}

// UpdateNotifierStateForSource update current notifier state for a given source.
func UpdateNotifierStateForSource(database moira.Database, clusterKey moira.ClusterKey, state *dto.NotifierState, clock moira.Clock) *api.ErrorResponse {
	err := database.SetNotifierStateForSource(clusterKey, moira.SelfStateActorManual, state.State, clock)
	if err != nil {
		return api.ErrorInternalServer(err)
	}

	return nil
}

// GetSystemSubscriptions returns system subscriptions matched to system tags.
func GetSystemSubscriptions(database moira.Database, systemTags []string) ([]*moira.SubscriptionData, *api.ErrorResponse) {
	subs, err := database.GetTagsSubscriptions(systemTags)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	return subs, nil
}
