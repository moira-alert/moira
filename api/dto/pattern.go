package dto

import (
	"net/http"
)

type PatternList struct {
	List []PatternData `json:"list"`
}

func (*PatternList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type PatternData struct {
	Metrics  []string       `json:"metrics"`
	Pattern  string         `json:"pattern"`
	Triggers []TriggerModel `json:"triggers"`
}
