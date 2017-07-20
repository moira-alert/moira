package handler

import (
	"github.com/go-chi/chi"
	"net/http"
)

func subscription(router chi.Router) {
	router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
		//Взять все подписки пользователя
	})

	router.Put("/", func(writer http.ResponseWriter, request *http.Request) {
		//todo какой-то check_json
		//Создать новую подписку
	})

	router.Route("/{SubscriptionId}", func(router chi.Router) {
		router.Delete("/", func(writer http.ResponseWriter, request *http.Request) {
			//Удалить подписку
		})

		router.Put("/test", func(writer http.ResponseWriter, request *http.Request) {
			//Выполнятеся после сохранения, высылает нотификацию сразу
		})
	})
}
