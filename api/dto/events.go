// nolint.
package dto

import (
	"net/http"

	"github.com/moira-alert/moira"
)

type EventsList struct {
	Page  int64                     `json:"page" example:"0" format:"int64"`
	Size  int64                     `json:"size" example:"100" format:"int64"`
	Total int64                     `json:"total" example:"10" format:"int64"`
	List  []moira.NotificationEvent `json:"list"`
}

func (*EventsList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
