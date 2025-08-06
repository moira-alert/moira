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

func ErrorMessageForSource(source moira.ClusterKey) string {
	return fmt.Sprintf("Something unexpected happened to Moira's %s metric source, so we temporarily turned off the notification mailing for it. We are already working on the problem and will fix it in the near future.", source.String())
}

type NotifierState struct {
	Actor   string `json:"actor" example:"AUTO"`
	State   string `json:"state" example:"ERROR"`
	Message string `json:"message,omitempty" example:"Moira has been turned off for maintenance"`
}

func (*NotifierState) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (state *NotifierState) Bind(r *http.Request) error {
	if state.State == "" {
		return fmt.Errorf("state can not be empty")
	}
	if state.State != moira.SelfStateOK && state.State != moira.SelfStateERROR {
		return fmt.Errorf("invalid state '%s'. State should be one of: <OK|ERROR>", state.State)
	}
	return nil
}

type NotifierStatesForSources struct {
	Sources map[string]NotifierState `json:"sources"`
}

func (*NotifierStatesForSources) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (states *NotifierStatesForSources) Bind(r *http.Request) error {
	for _, state := range states.Sources {
		if state.State == "" {
			return fmt.Errorf("state can not be empty")
		}
		if state.State != moira.SelfStateOK && state.State != moira.SelfStateERROR {
			return fmt.Errorf("invalid state '%s'. State should be one of: <OK|ERROR>", state.State)
		}
	}
	return nil
}
