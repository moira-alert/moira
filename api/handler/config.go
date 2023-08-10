package handler

import "net/http"

type Contact struct {
	Type  string `json:"type" example:"telegram"`
	Label string `json:"label" example:"Telegram"`
}

type ConfigurationResponse struct {
	RemoteAllowed bool      `json:"remoteAllowed" example:"false"`
	Contacts      []Contact `json:"contacts"`
}

// @Summary Get available configuration
// @ID get-web-config
// @Produce json
// @Success 200 {object} ConfigurationResponse "Configuration fetched successfully"
// @Router /api/config [get]
// @Tags config
func getWebConfig(configContent []byte) http.HandlerFunc {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.Write(configContent) //nolint
	})
}
