package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/beevee/go-chart"
	"github.com/go-chi/render"
	"github.com/go-graphite/carbonapi/date"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/plotting"
)

func renderTrigger(writer http.ResponseWriter, request *http.Request) {
	sourceProvider, targetName, from, to, triggerID, fetchRealtimeData, err := getEvaluationParameters(request)
	if err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err))
		return
	}
	metricsData, trigger, err := evaluateTargetMetrics(sourceProvider, from, to, triggerID, fetchRealtimeData)
	if err != nil {
		render.Render(writer, request, api.ErrorInternalServer(err))
		return
	}

	targetMetrics, ok := metricsData[targetName]
	if !ok {
		render.Render(writer, request, api.ErrorNotFound(fmt.Sprintf("Cannot find target %s", targetName)))
	}

	renderable, err := buildRenderable(request, trigger, targetMetrics, targetName)
	if err != nil {
		render.Render(writer, request, api.ErrorInternalServer(err))
		return
	}
	writer.Header().Set("Content-Type", "image/png")
	err = renderable.Render(chart.PNG, writer)
	if err != nil {
		render.Render(writer, request, api.ErrorInternalServer(fmt.Errorf("can not render plot %s", err.Error())))
	}
}

func getEvaluationParameters(request *http.Request) (sourceProvider *metricSource.SourceProvider, targetName string, from int64, to int64, triggerID string, fetchRealtimeData bool, err error) {
	sourceProvider = middleware.GetTriggerTargetsSourceProvider(request)
	targetName = middleware.GetTargetName(request)
	triggerID = middleware.GetTriggerID(request)
	fromStr := middleware.GetFromStr(request)
	toStr := middleware.GetToStr(request)
	from = date.DateParamToEpoch(fromStr, "UTC", 0, time.UTC)

	if from == 0 {
		return sourceProvider, "", 0, 0, "", false, fmt.Errorf("can not parse from: %s", fromStr)
	}
	from -= from % 60
	to = date.DateParamToEpoch(toStr, "UTC", 0, time.UTC)
	if to == 0 {
		return sourceProvider, "", 0, 0, "", false, fmt.Errorf("can not parse to: %s", fromStr)
	}
	realtime := request.URL.Query().Get("realtime")
	if realtime == "" {
		return
	}
	fetchRealtimeData, err = strconv.ParseBool(realtime)
	if err != nil {
		return sourceProvider, "", 0, 0, "", false, fmt.Errorf("invalid realtime param: %s", err.Error())
	}
	return
}

func evaluateTargetMetrics(metricSourceProvider *metricSource.SourceProvider, from, to int64, triggerID string, fetchRealtimeData bool) (map[string][]metricSource.MetricData, *moira.Trigger, error) {
	tts, trigger, err := controller.GetTriggerEvaluationResult(database, metricSourceProvider, from, to, triggerID, fetchRealtimeData)
	if err != nil {
		return nil, trigger, err
	}

	return tts, trigger, err
}

func buildRenderable(request *http.Request, trigger *moira.Trigger, metricsData []metricSource.MetricData, targetName string) (*chart.Chart, error) {
	timezone := request.URL.Query().Get("timezone")
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s timezone: %s", timezone, err.Error())
	}
	plotTheme := request.URL.Query().Get("theme")
	plotTemplate, err := plotting.GetPlotTemplate(plotTheme, location)
	if err != nil {
		return nil, fmt.Errorf("can not initialize plot theme %s", err.Error())
	}
	renderable, err := plotTemplate.GetRenderable(targetName, trigger, metricsData)
	if err != nil {
		return nil, err
	}
	return &renderable, err
}
