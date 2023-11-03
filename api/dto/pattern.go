// nolint
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
	Metrics  []string       `json:"metrics" example:"DevOps.my_server.hdd.freespace_mbytes, DevOps.my_server.hdd.freespace_mbytes, DevOps.my_server.db.*"`
	Pattern  string         `json:"pattern" example:"Devops.my_server.*"`
	Triggers []TriggerModel `json:"triggers"`
}
