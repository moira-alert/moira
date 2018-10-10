// nolint
package dto

import (
	"fmt"
	"net/http"
)

const (
	OK           = "OK"
	ERROR        = "ERROR"
	ErrorMessage = "Something unexpected happened to Moira, so we temporarily turned off the notification mailing. We are already working on the problem and will fix it in the near future."
)

type NotifierState struct {
	State   string `json:"state"`
	Message string `json:"message,omitempty"`
}

func (*NotifierState) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (contact *NotifierState) Bind(r *http.Request) error {
	if contact.State == "" {
		return fmt.Errorf("state can not be empty")
	}
	if contact.State != OK && contact.State != ERROR {
		return fmt.Errorf("invalid state '%s'. State should be one of: <OK|ERROR>", contact.State)
	}
	return nil
}
