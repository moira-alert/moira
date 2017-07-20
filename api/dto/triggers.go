package dto

import (
	"github.com/moira-alert/moira-alert"
	"net/http"
)

type TriggersList struct {
	Page  int64                     `json:"page"`
	Size  int64                     `json:"size"`
	Total int64                     `json:"total"`
	List  []moira.TriggerChecksData `json:"list"`
}

func (*TriggersList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
