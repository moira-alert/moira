package handler

import (
	"fmt"
	"net/http"
	"time"
	"strconv"

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

	tts, trigger, err := controller.GetTriggerEvaluationResult(database, int64(from), int64(to), triggerID)
	if err != nil {
		render.Render(writer, request, api.ErrorInternalServer(err))
		return
	}

	var metricsData = make([]*types.MetricData, 0, len(tts.Main)+len(tts.Additional))
	for _, ts := range tts.Main {
		metricsData = append(metricsData, &ts.MetricData)
	}
	for _, ts := range tts.Additional {
		metricsData = append(metricsData, &ts.MetricData)
	}

	var plot plotting.Plot
	font, _ := plotting.GetDefaultFont()
	isRaisingSet, raising, theme := getPlotParams(request)
	if !isRaisingSet {
		plot = plotting.FromParams(trigger.Name, theme, nil, trigger.WarnValue, trigger.ErrorValue)
	} else {
		plot = plotting.FromParams(trigger.Name, theme, &raising, trigger.WarnValue, trigger.ErrorValue)
	}
	renderable := plot.GetRenderable(metricsData, font)
	if len(renderable.Series) == 0 {
		render.Render(writer, request, api.ErrorInternalServer(fmt.Errorf("no timeseries found for %s", trigger.Name)))
		return
	}
	writer.Header().Set("Content-Type", "image/png")
	renderable.Render(chart.PNG, writer)
}

func getPlotParams(request *http.Request) (bool, bool, string) {
	theme := plotting.DarkTheme
	themeParam := request.URL.Query().Get("theme")
	raisingParam := request.URL.Query().Get("raising")
	if themeParam == "light" {
		theme = plotting.LightTheme
	}
	if raisingParam != "" {
		raising, err := strconv.ParseBool(raisingParam)
		if err == nil {
			return true, raising, theme
		}
	}
	return false, false, theme
}
