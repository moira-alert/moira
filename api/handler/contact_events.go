package handler

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"

	"github.com/ggicci/httpin"
	"github.com/go-chi/chi"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
)

func contactEvents(router chi.Router) {
	router.Route("/{contactId}", func(router chi.Router) {
		router.Use(middleware.ContactContext)
		router.Use(contactFilter)
		router.Use(httpin.NewInput(dto.EventIntervalQuery{}))
		router.Get("/", getContactByIdWithEvents)
	})
}

func getContactByIdWithEvents(writer http.ResponseWriter, request *http.Request) {
	contactData := request.Context().Value(contactKey).(moira.ContactData)
	eventQueryInterval := request.Context().Value(httpin.Input).(*dto.EventIntervalQuery)

	contactWithEvents, err := controller.GetContactByIdWithEventsLimit(
		database,
		contactData.ID,
		eventQueryInterval.From,
		eventQueryInterval.To)

	if err != nil {
		render.Render(writer, request, api.ErrorInternalServer(fmt.Errorf("can't fetch contact with %v and events", contactData.ID)))
	}

	if err := render.Render(writer, request, contactWithEvents); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}
