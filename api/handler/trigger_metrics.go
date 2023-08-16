package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/go-graphite/carbonapi/date"

	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
)

func triggerMetrics(router chi.Router) {
	router.With(middleware.DateRange("-10minutes", "now")).Get("/", getTriggerMetrics)
	router.Delete("/", deleteTriggerMetric)
	router.Delete("/nodata", deleteTriggerNodataMetrics)
}

// nolint: gofmt,goimports
//
//	@summary	Get metrics associated with certain trigger
//	@id			get-trigger-metrics
//	@tags		trigger
//	@produce	json
//	@param		triggerID	path		string							true	"Trigger ID"						default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		from		query		string							false	"Start time for metrics retrieval"	default(-10minutes)
//	@param		to			query		string							false	"End time for metrics retrieval"	default(now)
//	@success	200			{object}	dto.TriggerMetrics				"Trigger metrics retrieved successfully"
//	@failure	400			{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422			{object}	api.ErrorRenderExample			"Render error"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/trigger/{triggerID}/metrics [get]
func getTriggerMetrics(writer http.ResponseWriter, request *http.Request) {
	metricSourceProvider := middleware.GetTriggerTargetsSourceProvider(request)
	triggerID := middleware.GetTriggerID(request)
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

	triggerMetrics, err := controller.GetTriggerMetrics(database, metricSourceProvider, from, to, triggerID)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, triggerMetrics); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
	}
}

// nolint: gofmt,goimports
//
//	@summary	Delete metric from last check and all trigger pattern metrics
//	@id			delete-trigger-metric
//	@tags		trigger
//	@produce	json
//	@param		triggerID	path	string	true	"Trigger ID"				default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		name		query	string	true	"Name of the target metric"	default(DevOps.my_server.hdd.freespace_mbytes)
//	@success	200			"Trigger metric deleted successfully"
//	@failure	400			{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/trigger/{triggerID}/metrics [delete]
func deleteTriggerMetric(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)

	urlValues, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}

	metricName := urlValues.Get("name")
	if err := controller.DeleteTriggerMetric(database, metricName, triggerID); err != nil {
		render.Render(writer, request, err) //nolint
	}
}

// nolint: gofmt,goimports
//
//	@summary	Delete all metrics from last data which are in NODATA state. It also deletes all trigger patterns of those metrics
//	@id			delete-trigger-nodata-metrics
//	@tags		trigger
//	@produce	json
//	@param		triggerID	path	string	true	"Trigger ID"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200			"Trigger nodata metrics deleted successfully"
//	@failure	400			{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/trigger/{triggerID}/metrics/nodata [delete]
func deleteTriggerNodataMetrics(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	if err := controller.DeleteTriggerNodataMetrics(database, triggerID); err != nil {
		render.Render(writer, request, err) //nolint
	}
}
