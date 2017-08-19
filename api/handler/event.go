package handler

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/controller"
	"net/http"
)

func event(router chi.Router) {
	router.With(triggerContext, paginate(0, 100)).Get("/{triggerId}", func(writer http.ResponseWriter, request *http.Request) {
		context := request.Context()
		triggerId := context.Value("triggerId").(string)
		size := context.Value("size").(int64)
		page := context.Value("page").(int64)
		eventsList, err := controller.GetTriggerEvents(database, triggerId, page, size)
		if err != nil {
			render.Render(writer, request, err)
			return
		}
		if err := render.Render(writer, request, eventsList); err != nil {
			render.Render(writer, request, api.ErrorRender(err))
		}
	})
}
