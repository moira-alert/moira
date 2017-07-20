package handler

import (
	"github.com/go-chi/chi"
	"net/http"
)

func pattern(router chi.Router) {
	router.Get("/", getAllPatterns)
	router.Delete("/{pattern}", deletePattern)
}

func getAllPatterns(writer http.ResponseWriter, request *http.Request) {
	//Вытащить все паттерны по всем метрикам
}

func deletePattern(writer http.ResponseWriter, request *http.Request) {
	//удалить паттерн
	//todo не используется
}
