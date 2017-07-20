package dto

import (
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
