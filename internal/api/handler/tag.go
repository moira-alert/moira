package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/internal/api"
	"github.com/moira-alert/moira/internal/api/controller"
	"github.com/moira-alert/moira/internal/api/middleware"
)

func tag(router chi.Router) {
	router.Get("/", getAllTags)
	router.Get("/stats", getAllTagsAndSubscriptions)
	router.Route("/{tag}", func(router chi.Router) {
		router.Use(middleware.TagContext)
		router.Delete("/", removeTag)
	})
}

func getAllTags(writer http.ResponseWriter, request *http.Request) {
	tagData, err := controller.GetAllTags(database)
	if err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, tagData); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func getAllTagsAndSubscriptions(writer http.ResponseWriter, request *http.Request) {
	logger := middleware.GetLoggerEntry(request)
	data, err := controller.GetAllTagsAndSubscriptions(database, logger)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, data); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func removeTag(writer http.ResponseWriter, request *http.Request) {
	tagName := middleware.GetTag(request)
	response, err := controller.RemoveTag(database, tagName)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}
