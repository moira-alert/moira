package handler

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api/controller"
	"github.com/moira-alert/moira-alert/api/dto"
	"net/http"
)

func contact(router chi.Router) {
	router.Get("/", getAllContacts)
	router.Put("/", createNewContact)
	router.Delete("/{contactId}", deleteContact)
}

func getAllContacts(writer http.ResponseWriter, request *http.Request) {
	contacts, err := controller.GetAllContacts(database)
	if err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, contacts); err != nil {
		render.Render(writer, request, dto.ErrorRender(err))
		return
	}
}

func createNewContact(writer http.ResponseWriter, request *http.Request) {
	contact := &dto.Contact{}
	if err := render.Bind(request, contact); err != nil {
		render.Render(writer, request, dto.ErrorInvalidRequest(err))
		return
	}
	userLogin := request.Context().Value("login").(string)

	if err := controller.CreateContact(database, contact, userLogin); err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, contact); err != nil {
		render.Render(writer, request, dto.ErrorRender(err))
		return
	}
}

func deleteContact(writer http.ResponseWriter, request *http.Request) {
	contactId := chi.URLParam(request, "contactId")
	if contactId == "" {
		render.Render(writer, request, dto.ErrorInvalidRequest(fmt.Errorf("ContactId must be set")))
		return
	}
	userLogin := request.Context().Value("login").(string)

	err := controller.DeleteContact(database, contactId, userLogin)
	if err != nil {
		render.Render(writer, request, err)
	}
}
