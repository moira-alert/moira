package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
)

func event(router chi.Router) {
	router.With(middleware.TriggerContext, middleware.Paginate(0, 100)).Get("/{triggerId}", getEventsList)
	router.Delete("/all", deleteAllEvents)
}

func getEventsList(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	eventsList, err := controller.GetTriggerEvents(database, triggerID)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, eventsList); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
	}
}

func deleteAllEvents(writer http.ResponseWriter, request *http.Request) {
	if errorResponse := controller.DeleteAllEvents(database); errorResponse != nil {
		render.Render(writer, request, errorResponse)
	}
}
