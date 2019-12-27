// nolint
package dto

import (
	"net/http"

	"github.com/moira-alert/moira"
)

type TagsData struct {
	TagNames []string `json:"list"`
}

func (*TagsData) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type MessageResponse struct {
	Message string `json:"message"`
}

func (*MessageResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type TagsStatistics struct {
	List []TagStatistics `json:"list"`
}

type TagStatistics struct {
	TagName       string                   `json:"name"`
	Triggers      []string                 `json:"triggers"`
	Subscriptions []moira.SubscriptionData `json:"subscriptions"`
}

func (*TagsStatistics) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
