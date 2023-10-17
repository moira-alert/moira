package handler

import "net/http"

type ContactExample struct {
	Type        string `json:"type" example:"webhook kontur"`
	Label       string `json:"label" example:"Webhook Kontur"`
	Validation  string `json:"validation" example:"^(http|https):\\/\\/.*(testkontur.ru|kontur.host|skbkontur.ru)(:[0-9]{2,5})?\\/"`
	Placeholder string `json:"placeholder" example:"https://service.testkontur.ru/webhooks/moira"`
	Help        string `json:"help" example:"### Domains whitelist:\n - skbkontur.ru\n - testkontur.ru\n - kontur.host"`
}

type ConfigurationResponse struct {
	RemoteAllowed bool             `json:"remoteAllowed" example:"false"`
	Contacts      []ContactExample `json:"contacts"`
}

// nolint: gofmt,goimports
//
//	@summary	Get available configuration
//	@id			get-web-config
//	@tags		config
//	@produce	json
//	@success	200	{object}	ConfigurationResponse	"Configuration fetched successfully"
//	@router		/config [get]
func getWebConfig(configContent []byte) http.HandlerFunc {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.Write(configContent) //nolint
	})
}
