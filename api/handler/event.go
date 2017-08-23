package handler

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/controller"
	"github.com/moira-alert/moira-alert/api/middleware"
	"net/http"
)

func event(router chi.Router) {
	router.With(middleware.TriggerContext, middleware.Paginate(0, 100)).Get("/{triggerId}", func(writer http.ResponseWriter, request *http.Request) {
		triggerID := middleware.GetTriggerID(request)
		size := middleware.GetSize(request)
		page := middleware.GetPage(request)
		eventsList, err := controller.GetTriggerEvents(database, triggerID, page, size)
		if err != nil {
			render.Render(writer, request, err)
			return
		}
		if err := render.Render(writer, request, eventsList); err != nil {
			render.Render(writer, request, api.ErrorRender(err))
		}
	})
}
