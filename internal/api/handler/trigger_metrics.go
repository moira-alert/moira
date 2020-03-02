package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/go-graphite/carbonapi/date"

	"github.com/moira-alert/moira/internal/api"
	"github.com/moira-alert/moira/internal/api/controller"
	"github.com/moira-alert/moira/internal/api/middleware"
)

func triggerMetrics(router chi.Router) {
	router.With(middleware.DateRange("-10minutes", "now")).Get("/", getTriggerMetrics)
	router.Delete("/", deleteTriggerMetric)
	router.Delete("/nodata", deleteTriggerNodataMetrics)
}

func getTriggerMetrics(writer http.ResponseWriter, request *http.Request) {
	metricSourceProvider := middleware.GetTriggerTargetsSourceProvider(request)
	triggerID := middleware.GetTriggerID(request)
	fromStr := middleware.GetFromStr(request)
	toStr := middleware.GetToStr(request)
	from := date.DateParamToEpoch(fromStr, "UTC", 0, time.UTC)
	if from == 0 {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("can not parse from: %s", fromStr)))
		return
	}
	to := date.DateParamToEpoch(toStr, "UTC", 0, time.UTC)
	if to == 0 {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("can not parse to: %v", to)))
		return
	}
	triggerMetrics, err := controller.GetTriggerMetrics(database, metricSourceProvider, from, to, triggerID)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, triggerMetrics); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
	}
}

func deleteTriggerMetric(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	metricName := request.URL.Query().Get("name")
	if err := controller.DeleteTriggerMetric(database, metricName, triggerID); err != nil {
		render.Render(writer, request, err)
	}
}

func deleteTriggerNodataMetrics(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	if err := controller.DeleteTriggerNodataMetrics(database, triggerID); err != nil {
		render.Render(writer, request, err)
	}
}
