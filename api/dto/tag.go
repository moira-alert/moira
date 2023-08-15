// nolint
package dto

import (
	"net/http"

	"github.com/moira-alert/moira"
)

type TagsData struct {
	TagNames []string `json:"list" example:"cpu"`
}

func (*TagsData) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type MessageResponse struct {
	Message string `json:"message" example:"tag deleted"`
}

func (*MessageResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type TagsStatistics struct {
	List []TagStatistics `json:"list"`
}

type TagStatistics struct {
	TagName       string                   `json:"name" example:"cpu"`
	Triggers      []string                 `json:"triggers" example:"bcba82f5-48cf-44c0-b7d6-e1d32c64a88c"`
	Subscriptions []moira.SubscriptionData `json:"subscriptions"`
}

func (*TagsStatistics) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
