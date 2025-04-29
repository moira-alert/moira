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

// UpdateNotifierState update current notifier state.
func UpdateNotifierState(database moira.Database, state *dto.NotifierState) *api.ErrorResponse {
	err := database.SetNotifierState(moira.SelfStateActorManual, state.State)
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
