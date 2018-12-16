package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/go-graphite/carbonapi/date"
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/plotting"
	"github.com/moira-alert/moira/remote"
)

func renderTrigger(writer http.ResponseWriter, request *http.Request) {
	remoteCfg, from, to, triggerID, err := getEvaluationParameters(request)
	if err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err))
		return
	}
	metricsData, trigger, err := evaluateTriggerMetrics(remoteCfg, from, to, triggerID)
	if err != nil {
		render.Render(writer, request, api.ErrorInternalServer(err))
		return
	}
	renderable, err := buildRenderable(request, trigger, metricsData)
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

func getEvaluationParameters(request *http.Request) (remoteCfg *remote.Config, from int64, to int64, triggerID string, err error) {
	remoteCfg = middleware.GetRemoteConfig(request)
	triggerID = middleware.GetTriggerID(request)
	fromStr := middleware.GetFromStr(request)
	toStr := middleware.GetToStr(request)
	from = date.DateParamToEpoch(fromStr, "UTC", 0, time.UTC)
	if from == 0 {
		return remoteCfg, 0, 0, "", fmt.Errorf("can not parse from: %s", fromStr)
	}
	to = date.DateParamToEpoch(toStr, "UTC", 0, time.UTC)
	if to == 0 {
		return remoteCfg, 0, 0, "", fmt.Errorf("can not parse to: %s", fromStr)
	}
	return
}

func evaluateTriggerMetrics(remoteCfg *remote.Config, from, to int64, triggerID string) ([]*types.MetricData, *moira.Trigger, error) {
	tts, trigger, err := controller.GetTriggerEvaluationResult(database, remoteCfg, from, to, triggerID)
	if err != nil {
		return nil, trigger, err
	}
	var metricsData = make([]*types.MetricData, 0, len(tts.Main)+len(tts.Additional))
	for _, ts := range tts.Main {
		metricsData = append(metricsData, &ts.MetricData)
	}
	for _, ts := range tts.Additional {
		metricsData = append(metricsData, &ts.MetricData)
	}
	return metricsData, trigger, err
}

func buildRenderable(request *http.Request, trigger *moira.Trigger, metricsData []*types.MetricData) (*chart.Chart, error) {
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
	var metricsWhiteList = make([]string, 0)
	renderable := plotTemplate.GetRenderable(trigger, metricsData, metricsWhiteList)
	if len(renderable.Series) == 0 {
		return nil, plotting.ErrNoPointsToRender{TriggerName: trigger.Name}
	}
	return &renderable, err
}
