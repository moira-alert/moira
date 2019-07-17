package controller

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
)

// GetAPIState returns current API state
func GetAPIState() *dto.ServiceState {
	state := dto.ServiceState{State: "OK"}
	return &state
}

// GetNotifierState returns current notifier state
func GetNotifierState(database moira.Database) (*dto.ServiceState, *api.ErrorResponse) {
	state, err := database.GetNotifierState()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	notifierState := dto.ServiceState{State: state}
	if state == moira.SelfStateERROR {
		notifierState.Message = dto.ErrorMessage
	}

	return &notifierState, nil
}

// UpdateNotifierState update current notifier state
func UpdateNotifierState(database moira.Database, state *dto.ServiceState) *api.ErrorResponse {
	err := database.SetNotifierState(state.State)
	if err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}
