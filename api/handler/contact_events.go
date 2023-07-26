package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-graphite/carbonapi/date"

	"github.com/go-chi/render"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"

	"github.com/go-chi/chi"
	"github.com/moira-alert/moira/api/middleware"
)

func contactEvents(router chi.Router) {
	router.Route("/{contactId}/events", func(router chi.Router) {
		router.Use(middleware.ContactContext)
		router.Use(contactFilter)
		router.With(middleware.DateRange("-3hour", "now")).Get("/", getContactByIdWithEvents)
	})
}

func getContactByIdWithEvents(writer http.ResponseWriter, request *http.Request) {
	contactData := request.Context().Value(contactKey).(moira.ContactData)
	fromStr := middleware.GetFromStr(request)
	toStr := middleware.GetToStr(request)
	from := date.DateParamToEpoch(fromStr, "UTC", 0, time.UTC)
	if from == 0 {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("can not parse from: %s", fromStr))) //nolint
		return
	}
	to := date.DateParamToEpoch(toStr, "UTC", 0, time.UTC)
	if to == 0 {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("can not parse to: %v", to))) //nolint
		return
	}
	contactWithEvents, err := controller.GetContactEventsByIdWithLimit(database, contactData.ID, from, to)
	if err != nil {
		render.Render(writer, request, err)
	}
	if err := render.Render(writer, request, contactWithEvents); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}
