// nolint
package dto

import (
	"net/http"

	moira2 "github.com/moira-alert/moira/internal/moira"
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
	TagName       string                    `json:"name"`
	Triggers      []string                  `json:"triggers"`
	Subscriptions []moira2.SubscriptionData `json:"subscriptions"`
}

func (*TagsStatistics) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
