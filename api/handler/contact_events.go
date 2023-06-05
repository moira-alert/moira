package handler

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"

	"github.com/go-chi/chi"
	"github.com/moira-alert/moira/api/middleware"
)

func contactEvents(router chi.Router) {
	router.Route("/{contactId}", func(router chi.Router) {
		router.Use(middleware.ContactContext)
		router.Use(contactFilter)
		router.Get("/", getContactByIdWithEvents)
	})
}

func getContactByIdWithEvents(writer http.ResponseWriter, request *http.Request) {
	contactData := request.Context().Value(contactKey).(moira.ContactData)

	from := request.URL.Query().Get("from")
	to := request.URL.Query().Get("to")

	contactWithEvents, err := controller.GetContactByIdWithEventsLimit(database, contactData.ID, from, to)

	if err != nil {
		render.Render(writer, request, err)
	}

	if err := render.Render(writer, request, contactWithEvents); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}
