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
	router.Get("/stats", getTeamSubsStats)
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

// usersFilterForTeams is middleware that checks that user exists in this
func usersFilterForTeams(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		userID := middleware.GetLogin(request)
		teamID := middleware.GetTeamID(request)
		err := controller.CheckUserPermissionsForTeam(database, teamID, userID)
		if err != nil {
			render.Render(writer, request, err) //nolint
			return
		}
		next.ServeHTTP(writer, request)
	})
}

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

func getTeamSubsStats(writer http.ResponseWriter, request *http.Request) {
	stats, err := controller.GetTeamSubsStats(database, middleware.GetLoggerEntry(request))
	if err != nil {
		render.Render(writer, request, api.ErrorInternalServer(err)) //nolint
	}

	if err := render.Render(writer, request, stats); err != nil {
		render.Render(writer, request, api.ErrorRender(err)) //nolint
		return
	}
}
