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
		router.Delete("/", deleteTag)
		router.Put("/data", setTagMaintenance)
	})
}

func getAllTags(writer http.ResponseWriter, request *http.Request) {
	tagData, err := controller.GetAllTags(database)
	if err != nil {
		logger.Error(err.Err)
		if err := render.Render(writer, request, err); err != nil {
			render.Render(writer, request, dto.ErrorRender(err))
			return
		}
		return
	}

	if err := render.Render(writer, request, tagData); err != nil {
		render.Render(writer, request, dto.ErrorRender(err))
		return
	}
}

func getAllTagsAndSubscriptions(writer http.ResponseWriter, request *http.Request) {
	//вытащить все подписки по всем тегам
	//todo не используется
}

func deleteTag(writer http.ResponseWriter, request *http.Request) {
	//удалить tag к хуям
	//todo не используется
}

func setTagMaintenance(writer http.ResponseWriter, request *http.Request) {
	//todo какой-то check_json
	//Постим майтейнс для тега
}
