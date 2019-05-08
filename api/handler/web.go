package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
)

func web(сonfig *api.WebConfig) http.HandlerFunc {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		configContent, err := json.Marshal(&сonfig)
		if err != nil {
			render.Render(writer, request, api.ErrorInternalServer(err))
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.Write(configContent)
	})
}
