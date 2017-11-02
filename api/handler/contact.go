package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"

	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
)

func contact(router chi.Router) {
	router.Get("/", getAllContacts)
	router.Put("/", createNewContact)
	router.Route("/{contactId}", func(router chi.Router) {
		router.Use(middleware.ContactContext)
		router.Put("/", updateContact)
		router.Delete("/", removeContact)
		router.Post("/test", sendTestContactNotification)
	})
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

func updateContact(writer http.ResponseWriter, request *http.Request) {
	contact := &dto.Contact{}
	if err := render.Bind(request, contact); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err))
		return
	}
	contactID := middleware.GetContactID(request)
	userLogin := middleware.GetLogin(request)

	err := controller.CheckUserPermissionsForContact(database, contactID, userLogin)
	if err != nil {
		render.Render(writer, request, err)
	}

	if err := controller.UpdateContact(database, contact, contactID, userLogin); err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, contact); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func removeContact(writer http.ResponseWriter, request *http.Request) {
	contactID := middleware.GetContactID(request)
	userLogin := middleware.GetLogin(request)
	err := controller.CheckUserPermissionsForContact(database, contactID, userLogin)
	if err != nil {
		render.Render(writer, request, err)
	}

	err = controller.RemoveContact(database, contactID, userLogin)
	if err != nil {
		render.Render(writer, request, err)
	}
}

func sendTestContactNotification(writer http.ResponseWriter, request *http.Request) {
	contactID := middleware.GetContactID(request)
	userLogin := middleware.GetLogin(request)
	err := controller.CheckUserPermissionsForContact(database, contactID, userLogin)
	if err != nil {
		render.Render(writer, request, err)
	}

	err = controller.SendTestContactNotification(database, contactID, userLogin)
	if err != nil {
		render.Render(writer, request, err)
	}
}
