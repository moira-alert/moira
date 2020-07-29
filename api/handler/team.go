package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/moira-alert/moira/api/middleware"
)

func teams(router chi.Router) {
	router.Get("/", getAllTeams)
	router.Post("/", createTeam)
	router.Route("/{teamId}", func(router chi.Router) {
		router.Use(middleware.TeamContext)
		router.Get("/", getTeam)
		router.Patch("/", updateTeam)
		router.Route("/users", func(router chi.Router) {
			router.Get("/", getTeamUsers)
			router.Put("/", setTeamUsers)
			router.Post("/", addTeamUser)
			router.With(middleware.TeamUserIDContext).Delete("/{teamUserId}", deleteTeamUser)
		})
	})
}

func createTeam(writer http.ResponseWriter, request *http.Request) {

}

func getAllTeams(writer http.ResponseWriter, request *http.Request) {

}

func getTeam(writer http.ResponseWriter, request *http.Request) {

}

func updateTeam(writer http.ResponseWriter, request *http.Request) {

}

func getTeamUsers(writer http.ResponseWriter, request *http.Request) {

}

func setTeamUsers(writer http.ResponseWriter, request *http.Request) {

}

func addTeamUser(writer http.ResponseWriter, request *http.Request) {

}

func deleteTeamUser(writer http.ResponseWriter, request *http.Request) {

}
