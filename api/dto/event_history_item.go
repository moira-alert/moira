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
	List  []ContactEventItem `json:"list" binding:"required"`
	Page  int64              `json:"page" example:"0" format:"int64" binding:"required"`
	Size  int64              `json:"size" example:"100" format:"int64" binding:"required"`
	Total int64              `json:"total" example:"10" format:"int64" binding:"required"`
}

func (*ContactEventItemList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
