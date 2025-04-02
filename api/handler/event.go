package handler

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
)

func event(router chi.Router) {
	router.With(
		middleware.TriggerContext,
		middleware.Paginate(eventDefaultPage, eventDefaultSize),
		middleware.DateRange(eventDefaultFrom, eventDefaultTo),
		middleware.MetricContext(eventDefaultMetric),
		middleware.StatesContext(),
	).Get("/{triggerId}", getEventsList)
	router.With(middleware.AdminOnlyMiddleware()).Delete("/all", deleteAllEvents)
}

// nolint: gofmt,goimports
//
//	@summary	Gets all trigger events for current page and their count
//	@id			get-events-list
//	@tags		event
//	@produce	json
//	@param		triggerID	path		string							true	"The ID of updated trigger"																default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		size		query		int								false	"Number of items to be displayed on one page. if size = -1 then all events returned"	default(100)
//	@param		p			query		int								false	"Defines the number of the displayed page. E.g, p=2 would display the 2nd page"			default(0)
//	@param		from		query		string							false	"Start time of the time range"															default(-3hours)
//	@param		to			query		string							false	"End time of the time range"															default(now)
//	@param		metric		query		string							false	"Regular expression that will be used to filter events"									default(.*)
//	@param		states		query		[]string						false	"String of ',' separated state names. If empty then all states will be used."			collectionFormat(csv)
//	@success	200			{object}	dto.EventsList					"Events fetched successfully"
//	@Failure	400			{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@Failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@Failure	422			{object}	api.ErrorRenderExample			"Render error"
//	@Failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/event/{triggerID} [get]
func getEventsList(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	size := middleware.GetSize(request)
	page := middleware.GetPage(request)
	fromStr := middleware.GetFromStr(request)
	toStr := middleware.GetToStr(request)

	validator := DateRangeValidator{AllowInf: true}
	fromStr, toStr, err := validator.ValidateDateRangeStrings(fromStr, toStr)
	if err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}

	metricStr := middleware.GetMetric(request)
	metricRegexp, errCompile := regexp.Compile(metricStr)
	if errCompile != nil {
		_ = render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("can not parse metric \"%s\": %w", metricStr, errCompile)))
		return
	}

	states := middleware.GetStates(request)

	eventsList, errRsp := controller.GetTriggerEvents(database, triggerID, page, size, fromStr, toStr, metricRegexp, states)
	if err != nil {
		render.Render(writer, request, errRsp) //nolint
		return
	}
	if err := render.Render(writer, request, eventsList); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
	}
}

// nolint: gofmt,goimports
//
//	@summary	Deletes all notification events
//	@id			delete-all-events
//	@tags		event
//	@produce	json
//	@success	200	"Events removed successfully"
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/event/all [delete]
func deleteAllEvents(writer http.ResponseWriter, request *http.Request) {
	if errorResponse := controller.DeleteAllEvents(database); errorResponse != nil {
		render.Render(writer, request, errorResponse) //nolint
	}
}
