package dto

import (
	"github.com/moira-alert/moira-alert"
	"net/http"
)

type PatternList struct {
	List []Pattern `json:"list"`
}

func (*PatternList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type Pattern struct {
	Metrics  []string         `json:"metrics"`
	Pattern  string           `json:"pattern"`
	Triggers []*moira.Trigger `json:"triggers"`
}
