package controller

import (
	"time"

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
func UpdateNotifierState(database moira.Database, state *dto.NotifierState, now time.Time) *api.ErrorResponse {
	err := database.SetNotifierState(moira.NotifierState{
		Actor:     moira.SelfStateActorManual,
		State:     state.State,
		Timestamp: now,
	})
	if err != nil {
		return api.ErrorInternalServer(err)
	}

	return nil
}
