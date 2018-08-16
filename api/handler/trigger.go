package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/go-graphite/carbonapi/date"
	"github.com/moira-alert/moira/remote"

	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/expression"
	"github.com/moira-alert/moira/target"
)

func trigger(router chi.Router) {
	router.Use(middleware.TriggerContext)
	router.Put("/", updateTrigger)
	router.Get("/", getTrigger)
	router.Delete("/", removeTrigger)
	router.Get("/state", getTriggerState)
	router.Route("/throttling", func(router chi.Router) {
		router.Get("/", getTriggerThrottling)
		router.Delete("/", deleteThrottling)
	})
	router.Route("/metrics", func(router chi.Router) {
		router.With(middleware.DateRange("-10minutes", "now")).Get("/", getTriggerMetrics)
		router.Delete("/", deleteTriggerMetric)
	})
	router.Put("/maintenance", setMetricsMaintenance)
}

func updateTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	trigger := &dto.Trigger{}
	if err := render.Bind(request, trigger); err != nil {
		switch err.(type) {
		case target.ErrParseExpr, target.ErrEvalExpr, target.ErrUnknownFunction:
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("Invalid graphite targets: %s", err.Error())))
		case expression.ErrInvalidExpression:
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("Invalid expression: %s", err.Error())))
		case remote.ErrRemoteTriggerResponse:
			render.Render(writer, request, api.ErrorRemoteServerUnavailable(err))
		default:
			render.Render(writer, request, api.ErrorInternalServer(err))
		}
		return
	}

	timeSeriesNames := middleware.GetTimeSeriesNames(request)
	response, err := controller.UpdateTrigger(database, &trigger.TriggerModel, triggerID, timeSeriesNames)
	if err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func removeTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	err := controller.RemoveTrigger(database, triggerID)
	if err != nil {
		render.Render(writer, request, err)
	}
}

func getTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	if triggerID == "testlog" {
		panic("Test for multi line logs")
	}
	trigger, err := controller.GetTrigger(database, triggerID)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, trigger); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
	}
}

func getTriggerState(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	triggerState, err := controller.GetTriggerLastCheck(database, triggerID)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, triggerState); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
	}
}

func getTriggerThrottling(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	triggerState, err := controller.GetTriggerThrottling(database, triggerID)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, triggerState); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
	}
}

func deleteThrottling(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	err := controller.DeleteTriggerThrottling(database, triggerID)
	if err != nil {
		render.Render(writer, request, err)
	}
}

func getTriggerMetrics(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	fromStr := middleware.GetFromStr(request)
	toStr := middleware.GetToStr(request)
	from := date.DateParamToEpoch(fromStr, "UTC", 0, time.UTC)
	if from == 0 {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("Can not parse from: %s", fromStr)))
		return
	}
	to := date.DateParamToEpoch(toStr, "UTC", 0, time.UTC)
	if to == 0 {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("Can not parse to: %v", to)))
		return
	}
	triggerMetrics, err := controller.GetTriggerMetrics(database, from, to, triggerID)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, &triggerMetrics); err != nil {
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

func setMetricsMaintenance(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	metricsMaintenance := dto.MetricsMaintenance{}
	if err := render.Bind(request, &metricsMaintenance); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err))
		return
	}
	err := controller.SetMetricsMaintenance(database, triggerID, metricsMaintenance)
	if err != nil {
		render.Render(writer, request, err)
	}
}
