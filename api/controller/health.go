package controller

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
)

// GetNotifierState return current notifier state
func GetNotifierState(database moira.Database) (*dto.NotifierState, *api.ErrorResponse) {
	state, err := database.GetNotifierState()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	message, err  := database.GetNotifierMessage()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	notifierState := dto.NotifierState{State: state, Message: message}
	return &notifierState, nil
}

// UpdateNotifierState update current notifier state
func UpdateNotifierState(database moira.Database, state *dto.NotifierState) *api.ErrorResponse {
	err := database.SetNotifierState(state.State)
	if err != nil {
		return api.ErrorInternalServer(err)
	}

	if state.State == moira.SelfStateERROR && state.Message == "" {
		state.Message = dto.ErrorMessage
	}
	err = database.SetNotifierMessage(state.Message)
	if err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}
