package handler

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/go-graphite/carbonapi/date"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/controller"
	"github.com/moira-alert/moira-alert/api/dto"
	"github.com/moira-alert/moira-alert/checker"
	"net/http"
	"time"
)

func trigger(router chi.Router) {
	router.Use(triggerContext)
	router.Put("/", saveTrigger)
	router.Get("/", getTrigger)
	router.Delete("/", deleteTrigger)
	router.Get("/state", getTriggerState)
	router.Route("/throttling", func(router chi.Router) {
		router.Get("/", getTriggerThrottling)
		router.Delete("/", deleteThrottling)
	})
	router.Route("/metrics", func(router chi.Router) {
		router.With(dateRange("-10minutes", "now")).Get("/", getTriggerMetrics)
		router.Delete("/", deleteTriggerMetric)
	})
	router.Put("/maintenance", setMetricsMaintenance)
}

func saveTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerId := request.Context().Value("triggerId").(string)
	trigger := &dto.Trigger{}
	if err := render.Bind(request, trigger); err != nil {
		if _, ok := err.(checker.ErrInvalidExpression); ok || err == checker.ErrEvaluateTarget {
			render.Render(writer, request, api.ErrorInvalidRequest(err))
		} else {
			render.Render(writer, request, api.ErrorInternalServer(err))
		}
		return
	}

	timeSeriesNames := request.Context().Value("timeSeriesNames").(map[string]bool)
	response, err := controller.SaveTrigger(database, &trigger.Trigger, triggerId, timeSeriesNames)
	if err != nil {
		render.Render(writer, request, err)
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
		return
	}
}

func deleteTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerId := request.Context().Value("triggerId").(string)
	err := controller.DeleteTrigger(database, triggerId)
	if err != nil {
		render.Render(writer, request, err)
	}
}

func getTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerId := request.Context().Value("triggerId").(string)
	trigger, err := controller.GetTrigger(database, triggerId)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, trigger); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
	}
}

func getTriggerState(writer http.ResponseWriter, request *http.Request) {
	triggerId := request.Context().Value("triggerId").(string)
	triggerState, err := controller.GetTriggerLastCheck(database, triggerId)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, triggerState); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
	}
}

func getTriggerThrottling(writer http.ResponseWriter, request *http.Request) {
	triggerId := request.Context().Value("triggerId").(string)
	triggerState, err := controller.GetTriggerThrottling(database, triggerId)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, triggerState); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
	}
}

func deleteThrottling(writer http.ResponseWriter, request *http.Request) {
	triggerId := request.Context().Value("triggerId").(string)
	err := controller.DeleteTriggerThrottling(database, triggerId)
	if err != nil {
		render.Render(writer, request, err)
	}
}

func getTriggerMetrics(writer http.ResponseWriter, request *http.Request) {
	context := request.Context()
	triggerId := request.Context().Value("triggerId").(string)
	fromStr := context.Value("from").(string)
	toStr := context.Value("to").(string)
	from := date.DateParamToEpoch(fromStr, "UTC", 0, time.UTC)
	if from == 0 {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("Can not parse from: %s", fromStr)))
		return
	}
	to := date.DateParamToEpoch(toStr, "UTC", 0, time.UTC)
	if to == 0 {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("Can not parse to: %s", to)))
		return
	}
	triggerMetrics, err := controller.GetTriggerMetrics(database, int64(from), int64(to), triggerId)
	if err != nil {
		render.Render(writer, request, err)
		return
	}
	if err := render.Render(writer, request, &triggerMetrics); err != nil {
		render.Render(writer, request, api.ErrorRender(err))
	}
}

func deleteTriggerMetric(writer http.ResponseWriter, request *http.Request) {
	triggerId := request.Context().Value("triggerId").(string)
	metricName := request.URL.Query().Get("name")
	if metricName == "" {
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("Metric name can not be empty")))
		return
	}
	if err := controller.DeleteTriggerMetric(database, metricName, triggerId); err != nil {
		render.Render(writer, request, err)
	}
}

func setMetricsMaintenance(writer http.ResponseWriter, request *http.Request) {
	triggerId := request.Context().Value("triggerId").(string)
	metricsMaintenance := dto.MetricsMaintenance{}
	if err := render.Bind(request, &metricsMaintenance); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err))
		return
	}
	err := controller.SetMetricsMaintenance(database, triggerId, metricsMaintenance)
	if err != nil {
		render.Render(writer, request, err)
	}
}
