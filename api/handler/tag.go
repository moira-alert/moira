package handler

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/controller"
	"github.com/moira-alert/moira-alert/api/middleware"
	"net/http"
)

func tag(router chi.Router) {
	router.Get("/", getAllTags)
	router.Get("/stats", getAllTagsAndSubscriptions)
	router.Route("/{tag}", func(router chi.Router) {
		router.Use(tagContext)
		router.Delete("/", deleteTag)
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

func deleteTag(writer http.ResponseWriter, request *http.Request) {
	tagName := request.Context().Value("tag").(string)
	response, err := controller.DeleteTag(database, tagName)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}
