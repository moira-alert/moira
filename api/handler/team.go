package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/controller"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
)

func teams(router chi.Router) {
	router.Get("/", getAllTeams)
	router.Post("/", createTeam)
	router.Route("/{teamId}", func(router chi.Router) {
		router.Use(middleware.TeamContext)
		router.Use(usersFilterForTeams)
		router.Get("/", getTeam)
		router.Patch("/", updateTeam)
		router.Delete("/", deleteTeam)
		router.Route("/users", func(router chi.Router) {
			router.Get("/", getTeamUsers)
			router.Put("/", setTeamUsers)
			router.Post("/", addTeamUsers)
			router.With(middleware.TeamUserIDContext).Delete("/{teamUserId}", deleteTeamUser)
		})
		router.Get("/settings", getTeamSettings)
		router.Route("/subscriptions", teamSubscription)
		router.Route("/contacts", teamContact)
	})
}

// usersFilterForTeams is middleware that checks that user exists in this.
func usersFilterForTeams(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		userLogin := middleware.GetLogin(request)
		teamID := middleware.GetTeamID(request)
		auth := middleware.GetAuth(request)
		err := controller.CheckUserPermissionsForTeam(database, teamID, userLogin, auth)
		if err != nil {
			render.Render(writer, request, err) //nolint
			return
		}
		next.ServeHTTP(writer, request)
	})
}

// nolint: gofmt,goimports
//
//	@summary	Create a new team
//	@id			create-team
//	@tags		team
//	@accept		json
//	@produce	json
//	@param		team	body		dto.TeamModel					true	"Team data"
//	@success	200		{object}	dto.SaveTeamResponse			"Team created successfully"
//	@failure	400		{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	422		{object}	api.ErrorRenderExample			"Render error"
//	@failure	500		{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/teams [post]
func createTeam(writer http.ResponseWriter, request *http.Request) {
	user := middleware.GetLogin(request)
	team := dto.TeamModel{}
	err := render.Bind(request, &team)
	if err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint:errcheck
		return
	}
	response, apiErr := controller.CreateTeam(database, team, user)
	if apiErr != nil {
		render.Render(writer, request, apiErr) //nolint:errcheck
		return
	}
	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint:errcheck
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Get all teams
//	@id			get-all-teams
//	@tags		team
//	@produce	json
//	@success	200	{object}	dto.UserTeams					"Teams fetched successfully"
//	@failure	422	{object}	api.ErrorRenderExample			"Render error"
//	@failure	500	{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/teams [get]
func getAllTeams(writer http.ResponseWriter, request *http.Request) {
	user := middleware.GetLogin(request)
	response, err := controller.GetUserTeams(database, user)
	if err != nil {
		render.Render(writer, request, err) //nolint:errcheck
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint:errcheck
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Get a team by ID
//	@id			get-team
//	@tags		team
//	@produce	json
//	@param		teamID	path		string							true	"ID of the team"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200		{object}	dto.TeamModel					"Team updated successfully"
//	@failure	403		{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404		{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422		{object}	api.ErrorRenderExample			"Render error"
//	@failure	500		{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/teams/{teamID} [get]
func getTeam(writer http.ResponseWriter, request *http.Request) {
	teamID := middleware.GetTeamID(request)

	response, err := controller.GetTeam(database, teamID)
	if err != nil {
		render.Render(writer, request, err) //nolint:errcheck
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint:errcheck
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Update existing team
//	@id			update-team
//	@tags		team
//	@accept		json
//	@produce	json
//	@param		teamID	path		string							true	"ID of the team"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		team	body		dto.TeamModel					true	"Updated team data"
//	@success	200		{object}	dto.SaveTeamResponse			"Team updated successfully"
//	@failure	400		{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	403		{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404		{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422		{object}	api.ErrorRenderExample			"Render error"
//	@failure	500		{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/teams/{teamID} [patch]
func updateTeam(writer http.ResponseWriter, request *http.Request) {
	team := dto.TeamModel{}
	err := render.Bind(request, &team)
	if err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) //nolint:errcheck
		return
	}

	teamID := middleware.GetTeamID(request)

	response, apiErr := controller.UpdateTeam(database, teamID, team)
	if apiErr != nil {
		render.Render(writer, request, apiErr) //nolint:errcheck
		return
	}
	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint:errcheck
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Delete a team
//	@id			delete-team
//	@tags		team
//	@produce	json
//	@param		teamID	path		string							true	"ID of the team"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200		{object}	dto.SaveTeamResponse			"Team has been successfully deleted"
//	@failure	400		{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	403		{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404		{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422		{object}	api.ErrorRenderExample			"Render error"
//	@failure	500		{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/teams/{teamID} [delete]
func deleteTeam(writer http.ResponseWriter, request *http.Request) {
	userLogin := middleware.GetLogin(request)
	teamID := middleware.GetTeamID(request)

	response, apiErr := controller.DeleteTeam(database, teamID, userLogin)
	if apiErr != nil {
		render.Render(writer, request, apiErr) //nolint:errcheck
		return
	}
	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint:errcheck
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Get users of a team
//	@id			get-team-users
//	@tags		team
//	@produce	json
//	@param		teamID	path		string							true	"ID of the team"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200		{object}	dto.TeamMembers					"Users fetched successfully"
//	@failure	403		{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404		{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422		{object}	api.ErrorRenderExample			"Render error"
//	@failure	500		{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/teams/{teamID}/users [get]
func getTeamUsers(writer http.ResponseWriter, request *http.Request) {
	teamID := middleware.GetTeamID(request)

	response, err := controller.GetTeamUsers(database, teamID)
	if err != nil {
		render.Render(writer, request, err) // nolint:errcheck
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) // nolint:errcheck
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Set users of a team
//	@id			set-team-users
//	@tags		team
//	@accept		json
//	@produce	json
//	@param		teamID		path		string							true	"ID of the team"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		usernames	body		dto.TeamMembers					true	"Usernames to set as team members"
//	@success	200			{object}	dto.TeamMembers					"Team updated successfully"
//	@failure	400			{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	403			{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422			{object}	api.ErrorRenderExample			"Render error"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/teams/{teamID}/users [put]
func setTeamUsers(writer http.ResponseWriter, request *http.Request) {
	members := dto.TeamMembers{}
	err := render.Bind(request, &members)
	if err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) // nolint:errcheck
		return
	}

	teamID := middleware.GetTeamID(request)

	response, apiErr := controller.SetTeamUsers(database, teamID, members.Usernames)
	if err != nil {
		render.Render(writer, request, apiErr) // nolint:errcheck
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) // nolint:errcheck
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Add users to a team
//	@id			add-team-users
//	@tags		team
//	@accept		json
//	@produce	json
//	@param		teamID		path		string							true	"ID of the team"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		usernames	body		dto.TeamMembers					true	"Usernames to add to the team"
//	@success	200			{object}	dto.TeamMembers					"Team updated successfully"
//	@failure	400			{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	403			{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422			{object}	api.ErrorRenderExample			"Render error"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/teams/{teamID}/users [post]
func addTeamUsers(writer http.ResponseWriter, request *http.Request) {
	members := dto.TeamMembers{}
	err := render.Bind(request, &members)
	if err != nil {
		render.Render(writer, request, api.ErrorInvalidRequest(err)) // nolint:errcheck
		return
	}
	teamID := middleware.GetTeamID(request)

	response, apiErr := controller.AddTeamUsers(database, teamID, members.Usernames)
	if err != nil {
		render.Render(writer, request, apiErr) // nolint:errcheck
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) // nolint:errcheck
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Delete a user from a team
//	@id			delete-team-user
//	@tags		team
//	@produce	json
//	@param		teamID		path		string							true	"ID of the team"										default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@param		teamUserID	path		string							true	"User login in methods related to teams manipulation"	default(anonymous)
//	@success	200			{object}	dto.TeamMembers					"Team updated successfully"
//	@failure	400			{object}	api.ErrorInvalidRequestExample	"Bad request from client"
//	@failure	403			{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404			{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422			{object}	api.ErrorRenderExample			"Render error"
//	@failure	500			{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/teams/{teamID}/users/{teamUserID} [delete]
func deleteTeamUser(writer http.ResponseWriter, request *http.Request) {
	teamID := middleware.GetTeamID(request)
	userID := middleware.GetTeamUserID(request)

	response, err := controller.DeleteTeamUser(database, teamID, userID)
	if err != nil {
		render.Render(writer, request, err) // nolint:errcheck
		return
	}

	if err := render.Render(writer, request, response); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) // nolint:errcheck
		return
	}
}

// nolint: gofmt,goimports
//
//	@summary	Get team settings
//	@id			get-team-settings
//	@tags		team
//	@produce	json
//	@param		teamID	path		string							true	"ID of the team"	default(bcba82f5-48cf-44c0-b7d6-e1d32c64a88c)
//	@success	200		{object}	dto.TeamSettings				"Team settings"
//	@failure	403		{object}	api.ErrorForbiddenExample		"Forbidden"
//	@failure	404		{object}	api.ErrorNotFoundExample		"Resource not found"
//	@failure	422		{object}	api.ErrorRenderExample			"Render error"
//	@failure	500		{object}	api.ErrorInternalServerExample	"Internal server error"
//	@router		/teams/{teamID}/settings [get]
func getTeamSettings(writer http.ResponseWriter, request *http.Request) {
	teamID := middleware.GetTeamID(request)
	teamSettings, err := controller.GetTeamSettings(database, teamID)
	if err != nil {
		render.Render(writer, request, err) //nolint:errcheck
		return
	}

	if err := render.Render(writer, request, teamSettings); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint:errcheck
		return
	}
}
