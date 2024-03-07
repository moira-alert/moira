package handler

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
)

// nolint: gofmt,goimports.
//
//	@summary	Get web configuration.
//	@id			get-web-config.
//	@tags		config.
//	@produce	json.
//	@success	200	{object}	api.WebConfig			"Configuration fetched successfully".
//	@failure	422	{object}	api.ErrorRenderExample	"Render error".
//	@router		/config [get].
func getWebConfig(webConfig *api.WebConfig) http.HandlerFunc {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if err := render.Render(writer, request, webConfig); err != nil {
			render.Render(writer, request, api.ErrorRender(err)) //nolint
			return
		}
	})
}
