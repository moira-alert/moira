// nolint
package dto

import (
	"net/http"

	moira2 "github.com/moira-alert/moira/internal/moira"
)

type NotificationsList struct {
	Total int64                           `json:"total"`
	List  []*moira2.ScheduledNotification `json:"list"`
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
