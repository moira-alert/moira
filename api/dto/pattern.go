// nolint
package dto

import (
	"net/http"
)

type PatternList struct {
	List []PatternData `json:"list" binding:"required"`
}

func (*PatternList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type PatternData struct {
	Metrics  []string       `json:"metrics" binding:"required" example:"DevOps.my_server.hdd.freespace_mbytes, DevOps.my_server.hdd.freespace_mbytes, DevOps.my_server.db.*"`
	Pattern  string         `json:"pattern" binding:"required" example:"Devops.my_server.*"`
	Triggers []TriggerModel `json:"triggers" binding:"required"`
}
