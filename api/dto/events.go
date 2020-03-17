// nolint
package dto

import (
	"net/http"

	"github.com/moira-alert/moira"
)

type EventsList struct {
	Total int64                     `json:"total"`
	List  []moira.NotificationEvent `json:"list"`
}

func (*EventsList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
