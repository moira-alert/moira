// nolint
package dto

import (
	"net/http"

	"github.com/moira-alert/moira"
)

type EventsList struct {
	Page  int64                     `json:"page" example:"0" format:"int64" binding:"required"`
	Size  int64                     `json:"size" example:"100" format:"int64" binding:"required"`
	Total int64                     `json:"total" example:"10" format:"int64" binding:"required"`
	List  []moira.NotificationEvent `json:"list" binding:"required"`
}

func (*EventsList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
