package handler

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api/controller"
	"github.com/moira-alert/moira-alert/api/dto"
	"net/http"
)

func event(router chi.Router) {
	router.With(triggerContext, paginate(0, 100)).Get("/{triggerId}", func(writer http.ResponseWriter, request *http.Request) {
		context := request.Context()
		triggerId := context.Value("triggerId").(string)
		size := context.Value("size").(int64)
		page := context.Value("page").(int64)
		eventsList, err := controller.GetEvents(database, triggerId, page, size)
		if err != nil {
			render.Render(writer, request, err)
			return
		}
		if err := render.Render(writer, request, eventsList); err != nil {
			render.Render(writer, request, dto.ErrorRender(err))
		}
	})
}
