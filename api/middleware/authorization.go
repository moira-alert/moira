package middleware

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/moira-alert/moira/api"
)

// AdminOnlyMiddleware returns 403 if request for made by non-admin user.
func AdminOnlyMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			auth := GetAuth(r)
			userLogin := GetLogin(r)

			if auth.IsEnabled() && !auth.IsAdmin(userLogin) {
				render.Render(w, r, api.ErrorForbidden("Only administrators can use this")) //nolint:errcheck
				return
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
