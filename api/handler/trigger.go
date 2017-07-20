package handler

import (
	"github.com/go-chi/chi"
	"net/http"
)

func trigger(router chi.Router) {
	router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
		//дать триггер
	})

	router.Get("/state", func(writer http.ResponseWriter, request *http.Request) {
		//дать состояние триггера
	})

	router.Route("/throttling", func(router chi.Router) {
		router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
			//not found
		})

		router.Delete("/", func(writer http.ResponseWriter, request *http.Request) {
			//удалить throttling
		})
	})

	router.Route("/metrics", func(router chi.Router) {
		router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
			//not found
		})

		router.Delete("/", func(writer http.ResponseWriter, request *http.Request) {
			//?name=EG-FRONT-03.UserSettings.Read
		})
	})

	router.Put("/maintenance", func(writer http.ResponseWriter, request *http.Request) {
		//Установить maintenance
		//в body - время, до которого будет maintenance
		//Умеет в массив треггеров
	})
}
