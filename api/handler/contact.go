package handler

import (
	"github.com/go-chi/chi"
	"net/http"
)

func contact(router chi.Router) {
	router.Get("/", getAllContacts)
	router.Put("/", createNewContact)
	router.Delete("/{ContactId}", deleteContact)
}

func getAllContacts(writer http.ResponseWriter, request *http.Request) {
	//Дергает абсолютно все контакты
}

func createNewContact(writer http.ResponseWriter, request *http.Request) {
	//todo какой-то check_json
	//Создает новый контакт в админке пользователя
}

func deleteContact(writer http.ResponseWriter, request *http.Request) {
	//удалить контакт
}
