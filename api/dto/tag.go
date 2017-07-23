package dto

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"net/http"
)

type TagsData struct {
	TagNames []string                 `json:"list"`
	TagsMap  map[string]moira.TagData `json:"tags"`
}

func (*TagsData) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type Tag moira.TagData

func (tag *Tag) Bind(r *http.Request) error {
	if tag.Maintenance == nil {
		return fmt.Errorf("Tag maintenance can not be empty")
	}
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
	Data          moira.TagData            `json:"data"`
}

func (*TagsStatistics) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
