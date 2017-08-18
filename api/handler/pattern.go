package handler

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/controller"
	"net/http"
)

func pattern(router chi.Router) {
	router.Get("/", getAllPatterns)
	router.Delete("/{pattern}", deletePattern)
}

func getAllPatterns(writer http.ResponseWriter, request *http.Request) {
	patternsList, err := controller.GetAllPatterns(database)
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
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("Pattern must be set")))
		return
	}
	err := controller.DeletePattern(database, pattern)
	if err != nil {
		render.Render(writer, request, err)
	}
}
