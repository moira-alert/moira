package controller

import (
	"github.com/moira-alert/moira/internal/api"
	"github.com/moira-alert/moira/internal/api/dto"
	moira2 "github.com/moira-alert/moira/internal/moira"
)

// GetNotifierState return current notifier state
func GetNotifierState(database moira2.Database) (*dto.NotifierState, *api.ErrorResponse) {
	state, err := database.GetNotifierState()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	notifierState := dto.NotifierState{State: state}
	if state == moira2.SelfStateERROR {
		notifierState.Message = dto.ErrorMessage
	}

	return &notifierState, nil
}

// UpdateNotifierState update current notifier state
func UpdateNotifierState(database moira2.Database, state *dto.NotifierState) *api.ErrorResponse {
	err := database.SetNotifierState(state.State)
	if err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}
