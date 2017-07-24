package dto

import (
	"github.com/moira-alert/moira-alert"
	"net/http"
)

type TriggersList struct {
	Page  int64                 `json:"page"`
	Size  int64                 `json:"size"`
	Total int64                 `json:"total"`
	List  []moira.TriggerChecks `json:"list"`
}

func (*TriggersList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type Trigger struct {
	moira.Trigger
	Throttling int64 `json:"throttling"`
}

func (*Trigger) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type TriggerCheck struct {
	*moira.CheckData
	TriggerId string `json:"trigger_id"`
}

func (*TriggerCheck) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type MetricsMaintenance map[string]int64

func (*MetricsMaintenance) Bind(r *http.Request) error {
	return nil
}
