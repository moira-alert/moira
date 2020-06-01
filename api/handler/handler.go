package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/rs/cors"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	moiramiddle "github.com/moira-alert/moira/api/middleware"
)

var database moira.Database
var searchIndex moira.Searcher

const contactKey moiramiddle.ContextKey = "contact"
const subscriptionKey moiramiddle.ContextKey = "subscription"

// NewHandler creates new api handler request uris based on github.com/go-chi/chi
func NewHandler(db moira.Database, log moira.Logger, index moira.Searcher, config *api.Config, metricSourceProvider *metricSource.SourceProvider, webConfigContent []byte) http.Handler {
	database = db
	searchIndex = index
	router := chi.NewRouter()
	router.Use(render.SetContentType(render.ContentTypeJSON))
	router.Use(moiramiddle.UserContext)
	router.Use(moiramiddle.RequestLogger(log))
	router.Use(middleware.NoCache)

	router.NotFound(notFoundHandler)
	router.MethodNotAllowed(methodNotAllowedHandler)

	router.Route("/api", func(router chi.Router) {
		router.Use(moiramiddle.DatabaseContext(database))
		router.Get("/config", getWebConfig(webConfigContent))
		router.Route("/user", user)
		router.With(moiramiddle.Triggers(config.LocalMetricTTL, config.RemoteMetricTTL)).Route("/trigger", triggers(metricSourceProvider, searchIndex))
		router.Route("/tag", tag)
		router.Route("/pattern", pattern)
		router.Route("/event", event)
		router.Route("/contact", contact)
		router.Route("/subscription", subscription)
		router.Route("/notification", notification)
		router.Route("/health", health)
	})
	if config.EnableCORS {
		return cors.AllowAll().Handler(router)
	}
	return router
}

func notFoundHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(404) //nolint
	render.Render(writer, request, api.ErrNotFound) //nolint
}

func methodNotAllowedHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(405) //nolint
	render.Render(writer, request, api.ErrMethodNotAllowed) //nolint
}
