package handler

import (
	"net/http"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/support"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

func debug(router chi.Router) {
	triggerRoute := router.With(middleware.TriggerContext)
	triggerRoute.Get("/trigger/{triggerId}", pullTrigger)
	triggerRoute.Get("/trigger/{triggerId}/metrics", pullTriggerMetrics)
	triggerRoute.Put("/trigger/{triggerId}", pushTrigger)
	triggerRoute.Put("/trigger/{triggerId}/metrics", pushTriggerMetrics)
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

func pushTrigger(writer http.ResponseWriter, request *http.Request) {
	_, log := prepareTriggerContext(request)
	trigger := &moira.Trigger{}
	if err := render.DecodeJSON(request.Body, trigger); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}
	err := controller.PushTrigger(database, log, trigger)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
}

func pushTriggerMetrics(writer http.ResponseWriter, request *http.Request) {
	triggerID, log := prepareTriggerContext(request)
	metrics := []support.PatternMetrics{}
	if err := render.DecodeJSON(request.Body, &metrics); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}
	err := controller.PushTriggerMetrics(database, log, triggerID, metrics)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
}
