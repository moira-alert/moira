package handler

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api/dto"
	"net/http"
	"os"
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
	tagsNames, err := database.GetTagNames()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	tagsMap, err := database.GetTags(tagsNames)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	tagsData := dto.TagsData{
		TagNames: tagsNames,
		TagsMap:  tagsMap,
	}

	if err := render.Render(writer, request, &tagsData); err != nil {
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
