// nolint
package dto

import (
	"net/http"

	"github.com/moira-alert/moira"
)

type EventsList struct {
	Page  int64                     `json:"page"`
	Size  int64                     `json:"size"`
	Total int64                     `json:"total"`
	List  []moira.NotificationEvent `json:"list"`
}

type EventIntervalQuery struct {
	From uint64 `in:"query=from"`
	To   uint64 `in:"query=to"`
}

func (*EventsList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
