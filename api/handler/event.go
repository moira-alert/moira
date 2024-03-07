package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
)

func event(router chi.Router) {
	router.With(middleware.TriggerContext, middleware.Paginate(0, 100)).Get("/{triggerId}", getEventsList)
	router.Delete("/all", deleteAllEvents)
}

// nolint: gofmt,goimports.
//
//	@summary	Gets all trigger events for current page and their count.
//	@id			get-events-list.
//	@tags		event.
//	@produce	json.
//	@param		triggerID	path		string							true	"The ID of updated trigger"														default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c).
//	@param		size		query		int								false	"Number of items to be displayed on one page"									default(100).
//	@param		p			query		int								false	"Defines the number of the displayed page. E.g, p=2 would display the 2nd page"	default(0).
//	@success	200			{object}	dto.EventsList					"Events fetched successfully".
//	@Failure	400			{object}	api.ErrorInvalidRequestExample	"Bad request from client".
//	@Failure	404			{object}	api.ErrorNotFoundExample		"Resource not found".
//	@Failure	422			{object}	api.ErrorRenderExample			"Render error".
//	@Failure	500			{object}	api.ErrorInternalServerExample	"Internal server error".
//	@router		/event/{triggerID} [get].
func getEventsList(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	size := middleware.GetSize(request)
	page := middleware.GetPage(request)
	eventsList, err := controller.GetTriggerEvents(database, triggerID, page, size)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
	if err := render.Render(writer, request, eventsList); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
	}
}

// nolint: gofmt,goimports.
//
//	@summary	Deletes all notification events.
//	@id			delete-all-events.
//	@tags		event.
//	@produce	json.
//	@success	200	"Events removed successfully".
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error".
//	@router		/event/all [delete].
func deleteAllEvents(writer http.ResponseWriter, request *http.Request) {
	if errorResponse := controller.DeleteAllEvents(database); errorResponse != nil {
		render.Render(writer, request, errorResponse) //nolint
	}
}
