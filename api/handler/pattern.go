package handler

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
)

func pattern(router chi.Router) {
	router.Get("/", getAllPatterns)
	router.Delete("/{pattern}", deletePattern)
}

func getAllPatterns(writer http.ResponseWriter, request *http.Request) {
	logger := middleware.GetLoggerEntry(request)
	patternsList, err := controller.GetAllPatterns(database, logger)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, patternsList); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
	}
}

func deletePattern(writer http.ResponseWriter, request *http.Request) {
	pattern := chi.URLParam(request, "pattern")
	if pattern == "" {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("pattern must be set")))
		return
	}
	err := controller.DeletePattern(database, pattern)
	if err != nil {
		render.Render(writer, request, err)
	}
}
