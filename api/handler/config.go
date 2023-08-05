package handler

import "net/http"

// @Summary Get web configuration
// @Description View Moira's runtime configuration. For more details, see <https://moira.readthedocs.io/en/latest/installation/configuration.html>
// @ID get-web-config
// @Produce json
// @Success 200
// @Router /api/config [get]
// @Tags config
func getWebConfig(configContent []byte) http.HandlerFunc {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.Write(configContent) //nolint
	})
}
