package handler

import (
	"github.com/go-chi/chi"
	"net/http"
)

func notification(router chi.Router) {
	router.Get("/", getNotification)
	router.Delete("/", deleteNotification)
}

func getNotification(writer http.ResponseWriter, request *http.Request) {
	//todo хрен знает, что делает, очень похоже на то, что получает нотификаю, которую уже нужно отправить
}

func deleteNotification(writer http.ResponseWriter, request *http.Request) {
	//todo хрен знает, что делает, очень похоже на то, что удаляет нотификаю, которую уже нужно отправить
}
