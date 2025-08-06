package controller

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
)

// GetNotifierState return current notifier state.
func GetNotifierState(database moira.Database) (*dto.NotifierState, *api.ErrorResponse) {
	state, err := database.GetNotifierState()
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

// GetNotifierState return current notifier state.
func GetNotifierStatesForSources(database moira.Database) (*dto.NotifierStatesForSources, *api.ErrorResponse) {
	states, err := database.GetNotifierStateForSources()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	notifierStates := dto.NotifierStatesForSources{
		Sources: map[string]dto.NotifierState{},
	}

	for key, state := range states {
		notifierState := dto.NotifierState{
			Actor: state.Actor,
			State: state.State,
		}
		if state.State == moira.SelfStateERROR {
			notifierState.Message = dto.ErrorMessageForSource(key)
		}

		notifierStates.Sources[key.String()] = notifierState
	}

	return &notifierStates, nil
}

// UpdateNotifierState update current notifier state.
func UpdateNotifierState(database moira.Database, state *dto.NotifierState) *api.ErrorResponse {
	err := database.SetNotifierState(moira.SelfStateActorManual, state.State)
	if err != nil {
		return api.ErrorInternalServer(err)
	}

	return nil
}

// UpdateNotifierState update current notifier state.
func UpdateNotifierStateForSource(database moira.Database, clusterKey moira.ClusterKey, state *dto.NotifierState) *api.ErrorResponse {
	err := database.SetNotifierStateForSource(clusterKey, moira.SelfStateActorManual, state.State)
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
