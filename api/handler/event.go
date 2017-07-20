package handler

import (
	"github.com/go-chi/chi"
	"net/http"
)

func event(router chi.Router) {
	router.Get("/{triggerId}", func(writer http.ResponseWriter, request *http.Request) {
		//?p=0&size=100
		//взять эвенты по треггеру
	})
}
