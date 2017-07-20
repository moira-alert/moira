package handler

import (
	"github.com/go-chi/chi"
	"net/http"
)

func subscription(router chi.Router) {
	router.Get("/", getAllSubscriptions)
	router.Put("/", createSubscription)
	router.Route("/{SubscriptionId}", func(router chi.Router) {
		router.Delete("/", deleteSubscription)
		router.Put("/test", sendTestNotification)
	})
}

func getAllSubscriptions(writer http.ResponseWriter, request *http.Request) {
	//Взять все подписки пользователя
}

func createSubscription(writer http.ResponseWriter, request *http.Request) {
	//todo какой-то check_json
	//Создать новую подписку
}

func deleteSubscription(writer http.ResponseWriter, request *http.Request) {
	//Удалить подписку
}

func sendTestNotification(writer http.ResponseWriter, request *http.Request) {
	//Выполнятеся после сохранения, высылает нотификацию сразу
}
