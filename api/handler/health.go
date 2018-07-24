package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
)

func health(router chi.Router) {
	router.Get("/notifier", getNotifierState)
	router.Put("/notifier", setNotifierState)
}

func getNotifierState(writer http.ResponseWriter, request *http.Request) {
	state, err := controller.GetNotifierState(database)
	if err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, state); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func setNotifierState(writer http.ResponseWriter, request *http.Request) {
	state := &dto.NotifierState{}
	if err := render.Bind(request, state); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err))
		return
	}

	if err := controller.UpdateNotifierState(database, state); err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, state); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}

}
