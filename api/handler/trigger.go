package handler

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api/controller"
	"github.com/moira-alert/moira-alert/api/dto"
	"net/http"
)

func trigger(router chi.Router) {
	router.Get("/", getTrigger)
	router.Get("/state", getTriggerState)
	router.Route("/throttling", func(router chi.Router) {
		router.Get("/", getTriggerThrottling)
		router.Delete("/", deleteThrottling)
	})
	router.Route("/metrics", func(router chi.Router) {
		router.Get("/", getTriggerMetrics)
		router.Delete("/", deleteTriggerMetric)
	})
	router.Put("/maintenance", setMetricMaintenance)
}

func getTrigger(writer http.ResponseWriter, request *http.Request) {
	if triggerId := chi.URLParam(request, "triggerId"); triggerId != "" {
		trigger, err := controller.GetTrigger(database, triggerId)
		if err != nil {
			render.Render(writer, request, err)
			return
		}
		if err := render.Render(writer, request, trigger); err != nil {
			render.Render(writer, request, dto.ErrorRender(err))
		}
		return
	} else {
		render.Render(writer, request, dto.ErrorNotFound)
		return
	}
}

func getTriggerState(writer http.ResponseWriter, request *http.Request) {
	//дать состояние триггера
}

func getTriggerThrottling(writer http.ResponseWriter, request *http.Request) {
	//not found
}

func deleteThrottling(writer http.ResponseWriter, request *http.Request) {
	//удалить throttling
}

func getTriggerMetrics(writer http.ResponseWriter, request *http.Request) {
	//not found
}

func deleteTriggerMetric(writer http.ResponseWriter, request *http.Request) {
	//?name=EG-FRONT-03.UserSettings.Read
}

func setMetricMaintenance(writer http.ResponseWriter, request *http.Request) {
	//Установить maintenance
	//в body - время, до которого будет maintenance
	//Умеет в массив треггеров
}
