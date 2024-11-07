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
		router.With(
			middleware.DateRange(contactEventsDefaultFrom, contactEventsDefaultTo),
			middleware.Paginate(contactEventsDefaultPage, contactEventsDefaultSize),
		).Get("/", getContactEventHistoryByID)
	})
}

// nolint: gofmt,goimports
//
//	@summary	Get contact events by ID with time range
//	@id			get-contact-events-by-id
//	@tags		contact
//	@produce	json
//	@param		contactID	path		string							true	"Contact ID"																																	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		from		query		string							false	"Start time of the time range"																													default(-3hour)
//	@param		to			query		string							false	"End time of the time range"																													default(now)
//	@param		size		query		int								false	"Number of items to return or all items if size == -1 (if size == -1 p should be zero for correct work)"										default(100)
//	@param		p			query		int								false	"Defines the index of data portion (combined with size). E.g, p=2, size=100 will return records from 200 (including), to 300 (not including)"	default(0)
//	@success	200			{object}	dto.ContactEventItemList		"Successfully received contact events"
//	@failure	400			{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	403			{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422			{object}	api.ErrorRenderExample			"Render error"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/contact/{contactID}/events [get]
func getContactEventHistoryByID(writer http.ResponseWriter, request *http.Request) {
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

	contactWithEvents, err := controller.GetContactEventsHistoryByID(
		database,
		contactData.ID,
		from,
		to,
		middleware.GetPage(request),
		middleware.GetSize(request))
	if err != nil {
		render.Render(writer, request, err) //nolint
	}
	if err := render.Render(writer, request, contactWithEvents); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}
