// nolint
package dto

import (
	"net/http"

	"github.com/moira-alert/moira"
)

type NotificationsList struct {
	Total int64                          `json:"total"`
	List  []*moira.ScheduledNotification `json:"list"`
}

func (*NotificationsList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type NotificationDeleteResponse struct {
	Result int64 `json:"result"`
}

func (*NotificationDeleteResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
