package handler

import (
	"github.com/go-chi/chi"
	"net/http"
)

func pattern(router chi.Router) {
	router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
		//Вытащить все паттерны по всем метрикам
	})

	router.Delete("/{pattern}", func(writer http.ResponseWriter, request *http.Request) {
		//удалить паттерн
		//todo не используется
	})
}
