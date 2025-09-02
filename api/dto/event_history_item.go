package dto

import "net/http"

type ContactEventItem struct {
	TimeStamp int64  `json:"timestamp" format:"int64" binding:"required"`
	Metric    string `json:"metric" binding:"required"`
	State     string `json:"state" binding:"required"`
	OldState  string `json:"old_state" binding:"required"`
	TriggerID string `json:"trigger_id" binding:"required"`
}

type ContactEventItemList struct {
	List []ContactEventItem `json:"list" binding:"required"`
}

func (*ContactEventItemList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
