package handler

import (
	"net/http"
	"regexp"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/cmd"
)

func config(webConfig []byte, sentryConfig cmd.SentryConfig) func(chi.Router) {
	return func(router chi.Router) {
		router.Get("/", getWebConfig(webConfig))
		router.Get("/sentry", getSentryConfig(sentryConfig))
	}
}

type ContactExample struct {
	Type        string `json:"type" example:"webhook kontur"`
	Label       string `json:"label" example:"Webhook Kontur"`
	Validation  string `json:"validation" example:"^(http|https):\\/\\/.*(moira.ru)(:[0-9]{2,5})?\\/"`
	Placeholder string `json:"placeholder" example:"https://moira.ru/webhooks/moira"`
	Help        string `json:"help" example:"### Domains whitelist:\n - moira.ru\n"`
}

type ConfigurationResponse struct {
	RemoteAllowed bool             `json:"remoteAllowed" example:"false"`
	Contacts      []ContactExample `json:"contacts"`
}

// nolint: gofmt,goimports
//
//	@summary	Get contacts configuration
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

func isValidHost(allowedHostsRegex string, host string) bool {
	reg := regexp.MustCompile(allowedHostsRegex)
	return reg.Match([]byte(host))
}

// nolint: gofmt,goimports
//
//	@summary	Get sentry configuration
//	@id			get-sentry-config
//	@tags		config
//	@produce	json
//	@success	200	{object}	dto.SentryConfig			"Sentry configuration fetched successfully"
//	@failure	403	{object}	api.ErrorForbiddenExample	"Forbidden"
//	@failure	422	{object}	api.ErrorRenderExample		"Render error"
//	@router		/config/sentry [get]
func getSentryConfig(sentryConfig cmd.SentryConfig) http.HandlerFunc {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !isValidHost(sentryConfig.AllowedHostsRegex, request.Host) {
			render.Render(writer, request, api.ErrorForbidden("Not allowed")) //nolint
			return
		}

		config := dto.SentryConfig{
			DSN: sentryConfig.DSN,
		}

		if err := render.Render(writer, request, config); err != nil {
			render.Render(writer, request, api.ErrorRender(err)) //nolint
			return
		}
	})
}
