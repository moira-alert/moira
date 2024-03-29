// nolint
package dto

import (
	"net/http"

	"github.com/moira-alert/moira"
)

type NotificationsList struct {
	Total int64                          `json:"total" example:"0" format:"int64"`
	List  []*moira.ScheduledNotification `json:"list"`
}

func (*NotificationsList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type NotificationDeleteResponse struct {
	Result int64 `json:"result" example:"0" format:"int64"`
}

func (*NotificationDeleteResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
