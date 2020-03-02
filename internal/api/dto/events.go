// nolint
package dto

import (
	"net/http"

	moira2 "github.com/moira-alert/moira/internal/moira"
)

type EventsList struct {
	Page  int64                      `json:"page"`
	Size  int64                      `json:"size"`
	Total int64                      `json:"total"`
	List  []moira2.NotificationEvent `json:"list"`
}

func (*EventsList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
