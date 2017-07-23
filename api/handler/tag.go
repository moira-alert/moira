package handler

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api/controller"
	"github.com/moira-alert/moira-alert/api/dto"
	"net/http"
)

func tag(router chi.Router) {
	router.Get("/", getAllTags)
	router.Get("/stats", getAllTagsAndSubscriptions)
	router.Route("/{tag}", func(router chi.Router) {
		router.Use(tagContext)
		router.Delete("/", deleteTag)
		router.Put("/data", setTagMaintenance)
	})
}

func getAllTags(writer http.ResponseWriter, request *http.Request) {
	tagData, err := controller.GetAllTags(database)
	if err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, tagData); err != nil {
		render.Render(writer, request, dto.ErrorRender(err))
		return
	}
}

func getAllTagsAndSubscriptions(writer http.ResponseWriter, request *http.Request) {
	data, err := controller.GetAllTagsAndSubscriptions(database)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, data); err != nil {
		render.Render(writer, request, dto.ErrorRender(err))
		return
	}
}

func deleteTag(writer http.ResponseWriter, request *http.Request) {
	tagName := request.Context().Value("tag").(string)
	response, err := controller.DeleteTag(database, tagName)
	if err != nil {
		render.Render(writer, request, err)
	}
	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, dto.ErrorRender(err))
		return
	}
}

func setTagMaintenance(writer http.ResponseWriter, request *http.Request) {
	tag := &dto.Tag{}
	if err := render.Bind(request, tag); err != nil {
		render.Render(writer, request, dto.ErrorInvalidRequest(err))
		return
	}
	tagName := request.Context().Value("tag").(string)

	if err := controller.SetTagMaintenance(database, tagName, tag); err != nil {
		render.Render(writer, request, err)
	}
}
