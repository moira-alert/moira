// nolint
package dto

import (
	"fmt"
	"net/http"

	"github.com/moira-alert/moira"
)

const (
	ErrorMessage = "Something unexpected happened to Moira, so we temporarily turned off the notification mailing. We are already working on the problem and will fix it in the near future."
)

// ErrorMessageForSource constructs an error message for a given metric source.
func ErrorMessageForSource(source moira.ClusterKey) string {
	return fmt.Sprintf("Something unexpected happened to Moira's %s metric source, so we temporarily turned off the notification mailing for it. We are already working on the problem and will fix it in the near future.", source.String())
}

// NotifierState represents state of notifier: <OK|ERROR>.
type NotifierState struct {
	Actor   string `json:"actor" binding:"required" example:"AUTO"`
	State   string `json:"state" binding:"required" example:"ERROR"`
	Message string `json:"message,omitempty" example:"Moira has been turned off for maintenance"`
}

func (state *NotifierState) IsValid() bool {
	return state.State == moira.SelfStateOK || state.State == moira.SelfStateERROR
}

func (*NotifierState) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (state *NotifierState) Bind(r *http.Request) error {
	if state.State == "" {
		return fmt.Errorf("state can not be empty")
	}
	if !state.IsValid() {
		return fmt.Errorf("invalid state '%s'. State should be one of: <OK|ERROR>", state.State)
	}
	return nil
}

// NotifierState represents state of notifier for specific metric source: <OK|ERROR>.
type NotifierStateForSource struct {
	TriggerSource moira.TriggerSource `json:"trigger_source" binding:"required"`
	ClusterId     moira.ClusterId     `json:"cluster_id" binding:"required"`
	NotifierState
}

// NotifierState represents state of notifier for all metric sources: <OK|ERROR>.
type NotifierStatesForSources struct {
	Sources []NotifierStateForSource `json:"sources" binding:"required"`
}

func (*NotifierStatesForSources) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (states *NotifierStatesForSources) Bind(r *http.Request) error {
	for _, state := range states.Sources {
		if state.State == "" {
			return fmt.Errorf("state can not be empty")
		}
		if !state.IsValid() {
			return fmt.Errorf("invalid state '%s'. State should be one of: <OK|ERROR>", state.State)
		}
	}
	return nil
}
