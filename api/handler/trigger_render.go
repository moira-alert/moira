package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/go-graphite/carbonapi/date"
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart"

	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/plotting"
)

func renderTrigger(writer http.ResponseWriter, request *http.Request) {
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
		render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("can not parse to: %v", toStr)))
		return
	}

	tts, trigger, err := controller.GetTriggerEvaluationResult(database, from, to, triggerID)
	if err != nil {
		render.Render(writer, request, api.ErrorInternalServer(err))
		return
	}

	var metricsData = make([]*types.MetricData, 0, len(tts.Main)+len(tts.Additional))
	var metricsWhiteList = make([]string, 0)
	for _, ts := range tts.Main {
		metricsData = append(metricsData, &ts.MetricData)
	}
	for _, ts := range tts.Additional {
		metricsData = append(metricsData, &ts.MetricData)
	}

	plotTheme := request.URL.Query().Get("theme")
	plotTemplate, err := plotting.GetPlotTemplate(plotTheme)
	if err != nil {
		render.Render(writer, request, api.ErrorInternalServer(fmt.Errorf("can not initialize plot theme %s", err.Error())))
		return
	}
	renderable := plotTemplate.GetRenderable(trigger, metricsData, metricsWhiteList)
	if len(renderable.Series) == 0 {
		render.Render(writer, request, api.ErrorInternalServer(fmt.Errorf("no timeseries found for %s", trigger.Name)))
		return
	}
	writer.Header().Set("Content-Type", "image/png")
	err = renderable.Render(chart.PNG, writer)
	if err != nil {
		render.Render(writer, request, api.ErrorInternalServer(fmt.Errorf("can not render plot %s", err.Error())))
	}
}
