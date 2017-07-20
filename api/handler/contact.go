package handler

import (
	"github.com/go-chi/chi"
	"net/http"
)

func contact(router chi.Router) {
	router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
		//Дергает абсолютно все контакты
	})

	router.Put("/", func(writer http.ResponseWriter, request *http.Request) {
		//todo какой-то check_json
		//Создает новый контакт в админке пользователя
	})

	router.Delete("/{ContactId}", func(writer http.ResponseWriter, request *http.Request) {
		//удалить контакт
	})
}
