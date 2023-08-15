package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
)

func trigger(router chi.Router) {
	router.Use(middleware.TriggerContext)
	router.Put("/", updateTrigger)
	router.With(middleware.TriggerContext, middleware.Populate(false)).Get("/", getTrigger)
	router.Delete("/", removeTrigger)
	router.Get("/state", getTriggerState)
	router.Route("/throttling", func(router chi.Router) {
		router.Get("/", getTriggerThrottling)
		router.Delete("/", deleteThrottling)
	})
	router.Route("/metrics", triggerMetrics)
	router.Put("/setMaintenance", setTriggerMaintenance)
	router.With(middleware.DateRange("-1hour", "now")).With(middleware.TargetName("t1")).Get("/render", renderTrigger)
	router.Get("/dump", triggerDump)
}

// nolint: gofmt,goimports
//	@summary	Update existing trigger
//	@id			update-trigger
//	@tags		trigger
//	@produce	json
//	@param		x-webauth-user	header		string									false	"User session token"
//	@param		triggerID		path		string									true	"Trigger ID"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		body			body		dto.Trigger								true	"Trigger data"
//	@success	200				{object}	dto.SaveTriggerResponse					"Updated trigger"
//	@failure	400				{object}	api.ErrorInvalidRequestExample			"Bad request from client"
//	@failure	404				{object}	api.ErrorNotFoundExample				"Resource not found"
//	@failure	422				{object}	api.ErrorRenderExample					"Render error"
//	@failure	500				{object}	api.ErrorInternalServerExample			"Internal server error"
//	@failure	503				{object}	api.ErrorRemoteServerUnavailableExample	"Remote server unavailable"
//	@router		/trigger/{triggerID} [put]
func updateTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)

	trigger, err := getTriggerFromRequest(request)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	var problems []dto.TreeOfProblems
	if needValidate(request) {
		problems = validateTargets(request, trigger)
		if problems != nil && dto.DoesAnyTreeHaveError(problems) {
			writeErrorSaveResponse(writer, request, problems)
			return
		}
	}

	timeSeriesNames := middleware.GetTimeSeriesNames(request)
	response, err := controller.UpdateTrigger(database, &trigger.TriggerModel, triggerID, timeSeriesNames)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if problems != nil {
		response.CheckResult.Targets = problems
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}

func needValidate(request *http.Request) bool {
	const validateFlag = "validate"
	return request.URL.Query().Has(validateFlag)
}

// validateTargets checks targets of trigger.
// Returns tree of problems if there is any invalid child, else returns nil.
func validateTargets(request *http.Request, trigger *dto.Trigger) (problems []dto.TreeOfProblems) {
	ttl := getMetricTTLByTrigger(request, trigger)
	treesOfProblems := dto.TargetVerification(trigger.Targets, ttl, trigger.IsRemote)

	for _, tree := range treesOfProblems {
		if tree.TreeOfProblems != nil {
			return treesOfProblems
		}
	}

	return nil
}

func writeErrorSaveResponse(writer http.ResponseWriter, request *http.Request, treesOfProblems []dto.TreeOfProblems) {
	render.Status(request, http.StatusBadRequest)
	response := dto.SaveTriggerResponse{
		CheckResult: dto.TriggerCheckResponse{
			Targets: treesOfProblems,
		},
	}
	render.JSON(writer, request, response)
}

// nolint: gofmt,goimports
//	@summary	Remove trigger
//	@id			remove-trigger
//	@tags		trigger
//	@param		triggerID	path		string							true	"Trigger ID"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/trigger/{triggerID} [delete]
func removeTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	err := controller.RemoveTrigger(database, triggerID)
	if err != nil {
		render.Render(writer, request, err) //nolint
	}
}

// nolint: gofmt,goimports
//	@summary	Get an existing trigger
//	@id			get-trigger
//	@tags		trigger
//	@produce	json
//	@param		triggerID	path		string							true	"Trigger ID"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200			{object}	dto.Trigger						"Trigger data"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422			{object}	api.ErrorRenderExample			"Render error"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/trigger/{triggerID} [get]
func getTrigger(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)

	trigger, err := controller.GetTrigger(database, triggerID)
	if err != nil {
		render.Render(writer, request, err) //nolint
		return
	}

	if err := checkingTemplateFilling(request, *trigger); err != nil {
		middleware.GetLoggerEntry(request).Warning().
			Error(err.Err).
			Msg("Failed to check template")
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

// nolint: gofmt,goimports
//	@summary	Get the trigger state as at last check
//	@id			get-trigger-state
//	@tags		trigger
//	@produce	json
//	@param		triggerID	path		string							true	"Trigger ID"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200			{object}	dto.TriggerCheck				"Trigger state fetched successful"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422			{object}	api.ErrorRenderExample			"Render error"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/trigger/{triggerID}/state [get]
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

// nolint: gofmt,goimports
//	@summary	Get a trigger with its throttling i.e its next allowed message time
//	@id			get-trigger-throttling
//	@tags		trigger
//	@produce	json
//	@param		triggerID	path		string						true	"Trigger ID"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200			{object}	dto.ThrottlingResponse		"Trigger throttle info retrieved"
//	@failure	404			{object}	api.ErrorNotFoundExample	"Resource not found"
//	@failure	422			{object}	api.ErrorRenderExample		"Render error"
//	@router		/trigger/{triggerID}/throttling [get]
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

// nolint: gofmt,goimports
//	@summary	Deletes throttling for a trigger
//	@id			delete-trigger-throttling
//	@tags		trigger
//	@param		triggerID	path	string	true	"Trigger ID"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200			"Trigger throttling has been deleted"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/trigger/{triggerID}/throttling [delete]
func deleteThrottling(writer http.ResponseWriter, request *http.Request) {
	triggerID := middleware.GetTriggerID(request)
	err := controller.DeleteTriggerThrottling(database, triggerID)
	if err != nil {
		render.Render(writer, request, err) //nolint
	}
}

// nolint: gofmt,goimports
//	@summary	Set metrics and the trigger itself to maintenance mode
//	@id			set-trigger-maintenance
//	@tags		trigger
//	@produce	json
//	@param		triggerID		path	string					true	"Trigger ID"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		body			body	dto.TriggerMaintenance	true	"Maintenance data"
//	@param		x-webauth-user	header	string					false	"User session token"
//	@success	200				"Trigger or metric have been scheduled for maintenance"
//	@failure	400				{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	404				{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	500				{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/trigger/{triggerID}/setMaintenance [put]
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

// nolint: gofmt,goimports
//	@summary	Get trigger dump
//	@id			get-trigger-dump
//	@tags		trigger
//	@produce	json
//	@param		triggerID	path		string							true	"Trigger ID"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200			{object}	dto.TriggerDump					"Trigger dump"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/trigger/{triggerID}/dump [get]
func triggerDump(writer http.ResponseWriter, request *http.Request) {
	triggerID, log := prepareTriggerContext(request)

	if dump, err := controller.GetTriggerDump(database, log, triggerID); err != nil {
		render.Render(writer, request, err) //nolint
	} else {
		render.JSON(writer, request, dump)
	}
}

func prepareTriggerContext(request *http.Request) (triggerID string, log moira.Logger) {
	logger := middleware.GetLoggerEntry(request)
	triggerID = middleware.GetTriggerID(request)
	log = logger.Clone().String(moira.LogFieldNameTriggerID, triggerID)
	return triggerID, log
}
