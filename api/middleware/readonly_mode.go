package middleware

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
)

func ReadOnlyMiddleware(config *api.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if config.Flags.IsReadonlyEnabled && IsMutatingMethod(r.Method) {
				render.Render(w, r, api.ErrorForbidden("Moira is currently in read-only mode")) //nolint:errcheck
				return
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func IsMutatingMethod(method string) bool {
	return method == http.MethodPut || method == http.MethodPost || method == http.MethodPatch
}
