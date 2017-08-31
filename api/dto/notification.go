// nolint
package dto

import (
	"github.com/moira-alert/moira-alert"
	"net/http"
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
