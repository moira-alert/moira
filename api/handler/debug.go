package handler

import (
	"net/http"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

func debug(router chi.Router) {
	router.With(middleware.TriggerContext).Get("/trigger/{triggerId}", pullTrigger)
	router.With(middleware.TriggerContext).Get("/trigger/{triggerId}/metrics", pullTriggerMetrics)
}

func prepareTriggerContext(request *http.Request) (triggerID string, log moira.Logger) {
	logger := middleware.GetLoggerEntry(request)
	triggerID = middleware.GetTriggerID(request)
	log = logger.Clone().String(moira.LogFieldNameTriggerID, triggerID)
	return triggerID, log
}

func pullTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerID, log := prepareTriggerContext(request)
	trigger, err := controller.PullTrigger(database, log, triggerID)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
	render.JSON(writer, request, &trigger)
}

func pullTriggerMetrics(writer http.ResponseWriter, request *http.Request) {
	triggerID, log := prepareTriggerContext(request)
	metrics, err := controller.PullTriggerMetrics(database, log, triggerID)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
	render.JSON(writer, request, &metrics)
}
