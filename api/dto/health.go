// nolint
package dto

import (
	"fmt"
	"net/http"

	"github.com/moira-alert/moira"
)

type NotifierState struct {
	State   string `json:"state"`
	Message string `json:"message,omitempty"`
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
