package handler

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api/controller"
	"github.com/moira-alert/moira-alert/api/dto"
	"net/http"
)

func event(router chi.Router) {
	router.Get("/{triggerId}", func(writer http.ResponseWriter, request *http.Request) {
		if triggerId := chi.URLParam(request, "triggerId"); triggerId != "" {
			page, size := getPageAndSize(request, 0, 100)
			eventsList, err := controller.GetEvents(database, triggerId, page, size)
			if err != nil {
				logger.Error(err.Err)
				render.Render(writer, request, err)
				return
			}
			if err := render.Render(writer, request, eventsList); err != nil {
				render.Render(writer, request, dto.ErrorRender(err))
			}
		} else {
			render.Render(writer, request, dto.ErrorNotFound)
			return
		}
	})
}
