package dto

import "net/http"

type ContactEventItem struct {
	TimeStamp int64  `json:"timestamp"`
	Metric    string `json:"metric"`
	State     string `json:"state"`
	OldState  string `json:"old_state"`
	TriggerID string `json:"trigger_id"`
}

type ContactEventItemList struct {
	List []ContactEventItem `json:"list"`
}

func (*ContactEventItemList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
