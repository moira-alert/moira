package middleware

// The original work was derived from Goji's middleware, source:
// https://github.com/zenazn/goji/tree/master/web/middleware

import (
	"fmt"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert/api"
	"net/http"
	"os"
	"runtime/debug"
)

// Recoverer is a middleware that recovers from panics, logs the panic (and a
// backtrace), and returns a HTTP 500 (Internal Server Error) status if
// possible. Recoverer prints a request ID if one is provided.
//
// Alternatively, look at https://github.com/pressly/lg middleware pkgs.
func Recoverer(next http.Handler) http.Handler {
	fn := func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {

				logEntry := middleware.GetLogEntry(request)
				if logEntry != nil {
					logEntry.Panic(rvr, debug.Stack())
				} else {
					fmt.Fprintf(os.Stderr, "Panic: %+v\n", rvr)
					debug.PrintStack()
				}

				render.Render(writer, request, api.ErrorInternalServer(fmt.Errorf("Internal Server Error")))
			}
		}()

		next.ServeHTTP(writer, request)
	}

	return http.HandlerFunc(fn)
}
