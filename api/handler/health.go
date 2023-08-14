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

// @summary	Get notifier state
// @id			get-notifier-state
// @tags		health
// @produce	json
// @success	200	{object}	dto.NotifierState				"Notifier state retrieved"
// @failure	422	{object}	api.ErrorRenderExample			"Render error"
// @failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
// @router		/health/notifier [get]
func getNotifierState(writer http.ResponseWriter, request *http.Request) {
	state, err := controller.GetNotifierState(database)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, state); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// @summary	Update notifier state
// @id			set-notifier-state
// @tags		health
// @accept		json
// @produce	json
// @param		state	body		dto.NotifierState				true	"New notifier state"
// @success	200		{object}	dto.NotifierState				"Update state of the Moira service"
// @failure	400		{object}	api.ErrorInvalidRequestExample	"Bad request from client"
// @failure	422		{object}	api.ErrorRenderExample			"Render error"
// @failure	500		{object}	api.ErrorInternalServerExample	"Internal server error"
// @router		/health/notifier [put]
func setNotifierState(writer http.ResponseWriter, request *http.Request) {
	state := &dto.NotifierState{}
	if err := render.Bind(request, state); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}

	if err := controller.UpdateNotifierState(database, state); err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, state); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}
