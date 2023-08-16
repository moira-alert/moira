package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
)

func tag(router chi.Router) {
	router.Get("/", getAllTags)
	router.Get("/stats", getAllTagsAndSubscriptions)
	router.Route("/{tag}", func(router chi.Router) {
		router.Use(middleware.TagContext)
		router.Delete("/", removeTag)
	})
}

// nolint: gofmt,goimports
//
//	@summary	Get all tags
//	@id			get-all-tags
//	@tags		tag
//	@produce	json
//	@success	200	{object}	dto.TagsData					"Tags fetched successfully"
//	@failure	422	{object}	api.ErrorRenderExample			"Render error"
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/tag [get]
func getAllTags(writer http.ResponseWriter, request *http.Request) {
	tagData, err := controller.GetAllTags(database)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, tagData); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Get all tags and their subscriptions
//	@id			get-all-tags-and-subscriptions
//	@tags		tag
//	@produce	json
//	@success	200	{object}	dto.TagsStatistics				"Successful"
//	@failure	422	{object}	api.ErrorRenderExample			"Render error"
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/tag/stats [get]
func getAllTagsAndSubscriptions(writer http.ResponseWriter, request *http.Request) {
	logger := middleware.GetLoggerEntry(request)
	data, err := controller.GetAllTagsAndSubscriptions(database, logger)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
	if err := render.Render(writer, request, data); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Remove a tag
//	@id			remove-tag
//	@tags		tag
//	@produce	json
//	@param		tag	path		string							true	"Name of the tag to remove"	default(cpu)
//	@success	200	{object}	dto.MessageResponse				"Tag removed successfully"
//	@failure	400	{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	422	{object}	api.ErrorRenderExample			"Render error"
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/tag/{tag} [delete]
func removeTag(writer http.ResponseWriter, request *http.Request) {
	tagName := middleware.GetTag(request)
	response, err := controller.RemoveTag(database, tagName)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}
