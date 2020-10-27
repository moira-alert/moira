package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/metric_source/remote"

	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/expression"
)

func trigger(router chi.Router) {
	router.Use(middleware.TriggerContext)
	router.Put("/", updateTrigger)
	router.With(middleware.TriggerContext,
		middleware.Populate(false)).Get("/", getTrigger)
	router.Delete("/", removeTrigger)
	router.Get("/state", getTriggerState)
	router.Route("/throttling", func(router chi.Router) {
		router.Get("/", getTriggerThrottling)
		router.Delete("/", deleteThrottling)
	})
	router.Route("/metrics", triggerMetrics)
	router.Put("/setMaintenance", setTriggerMaintenance)
	router.With(middleware.DateRange("-1hour", "now")).With(middleware.TargetName("t1")).Get("/render", renderTrigger)
}

func updateTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	trigger := &dto.Trigger{}

	if err := render.Bind(request, trigger); err != nil {
		switch err := err.(type) {
		case local.ErrParseExpr, local.ErrEvalExpr, local.ErrUnknownFunction:
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("invalid graphite targets: %s", err.Error()))) //nolint
		case expression.ErrInvalidExpression:
			render.Render(writer, request, api.ErrorInvalidRequest(fmt.Errorf("invalid expression: %s", err.Error()))) //nolint
		case api.ErrInvalidRequestContent:
			render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		case remote.ErrRemoteTriggerResponse:
			response := api.ErrorRemoteServerUnavailable(err)
			middleware.GetLoggerEntry(request).Error("%s : %s : %s", response.StatusText, response.ErrorText, err.Target)
			render.Render(writer, request, response) //nolint
		default:
			render.Render(writer, request, api.ErrorInternalServer(err)) //nolint
		}

		if err := checkingTemplateFilling(request, *trigger); err != nil {
			render.Render(writer, request, err) //nolint
		}

		return
	}

	timeSeriesNames := middleware.GetTimeSeriesNames(request)
	response, err := controller.UpdateTrigger(database, &trigger.TriggerModel, triggerID, timeSeriesNames)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

func removeTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	err := controller.RemoveTrigger(database, triggerID)
	if err != nil {
		render.Render(writer, request, err) //nolint
	}
}

func getTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	if triggerID == "testlog" {
		panic("Test for multi line logs")
	}

	trigger, err := controller.GetTrigger(database, triggerID)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := checkingTemplateFilling(request, *trigger); err != nil {
		middleware.GetLoggerEntry(request).Warning(err)
	}

	if err := render.Render(writer, request, trigger); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
	}
}

func checkingTemplateFilling(request *http.Request, trigger dto.Trigger) *api.ErrorResponse {
	if !middleware.GetPopulated(request) {
		return nil
	}

	eventsList, err := controller.GetTriggerEvents(database, trigger.ID, 0, 3)
	if err != nil {
		return err
	}

	if err := trigger.PopulatedDescription(eventsList.List); err != nil {
		return api.ErrorRender(err)
	}

	return nil
}

func getTriggerState(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	triggerState, err := controller.GetTriggerLastCheck(database, triggerID)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
	if err := render.Render(writer, request, triggerState); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
	}
}

func getTriggerThrottling(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	triggerState, err := controller.GetTriggerThrottling(database, triggerID)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}
	if err := render.Render(writer, request, triggerState); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
	}
}

func deleteThrottling(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	err := controller.DeleteTriggerThrottling(database, triggerID)
	if err != nil {
		render.Render(writer, request, err) //nolint
	}
}

func setTriggerMaintenance(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	triggerMaintenance := dto.TriggerMaintenance{}
	if err := render.Bind(request, &triggerMaintenance); err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint
		return
	}
	userLogin := middleware.GetLogin(request)
	timeCallMaintenance := time.Now().Unix()

	err := controller.SetTriggerMaintenance(database, triggerID, triggerMaintenance, userLogin, timeCallMaintenance)
	if err != nil {
		render.Render(writer, request, err) //nolint
	}
}
