package handler

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	"net/http"
)

func contact(router chi.Router) {
	router.Get("/", getAllContacts)
	router.Put("/", createNewContact)
	router.Delete("/{contactId}", removeContact)
	router.Post("/{contactId}/test", testContact)
}

func getAllContacts(writer http.ResponseWriter, request *http.Request) {
	contacts, err := controller.GetAllContacts(database)
	if err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, contacts); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func createNewContact(writer http.ResponseWriter, request *http.Request) {
	contact := &dto.Contact{}
	if err := render.Bind(request, contact); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err))
		return
	}
	userLogin := middleware.GetLogin(request)

	if err := controller.CreateContact(database, contact, userLogin); err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, contact); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func removeContact(writer http.ResponseWriter, request *http.Request) {
	contactID := chi.URLParam(request, "contactId")
	if contactID == "" {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("ContactId must be set")))
		return
	}
	userLogin := middleware.GetLogin(request)

	err := controller.RemoveContact(database, contactID, userLogin)
	if err != nil {
		render.Render(writer, request, err)
	}
}

func testContact(writer http.ResponseWriter, request *http.Request) {
	contactID := chi.URLParam(request, "contactId")
	if contactID == "" {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("ContactId must be set")))
		return
	}

	err := controller.TestContact(database, contactID)
	if err != nil {
		render.Render(writer, request, err)
	}
}
