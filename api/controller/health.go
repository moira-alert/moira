package controller

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
)

// GetNotifierState return current notifier state
func GetNotifierState(database moira.Database) (*dto.NotifierState, *api.ErrorResponse) {
	state, message, err := database.GetNotifierState()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	notifierState := dto.NotifierState{State: state, Message: message}
	if state == moira.SelfStateERROR && message == "" {
		notifierState.Message = moira.SelfStateErrorMessage
	}

	return &notifierState, nil
}

// UpdateNotifierState update current notifier state
func UpdateNotifierState(database moira.Database, state *dto.NotifierState) *api.ErrorResponse {
	if state.State == moira.SelfStateERROR && state.Message == "" {
		state.Message = moira.SelfStateErrorMessage
	}
	err := database.SetNotifierState(state.State, state.Message)
	if err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}
