package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"
)

// ErrorResponse represents custom error response with statusText and error description
type ErrorResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	ErrorText  string `json:"error,omitempty"` // application-level error message, for debugging
}

// Render realization method for render.renderer
func (e *ErrorResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

// ErrorInternalServer returns error response with status=500 and given error
func ErrorInternalServer(err error) *ErrorResponse {
	return &ErrorResponse{
		Err:            err,
		HTTPStatusCode: http.StatusInternalServerError,
		StatusText:     "Internal Server Error",
		ErrorText:      err.Error(),
	}
}

// ErrorInvalidRequest return error response with status = 400 and given error
func ErrorInvalidRequest(err error) *ErrorResponse {
	return &ErrorResponse{
		Err:            err,
		HTTPStatusCode: http.StatusBadRequest,
		StatusText:     "Invalid request",
		ErrorText:      err.Error(),
	}
}

// ErrorRender return 422 render error and used for response rendering errors
func ErrorRender(err error) *ErrorResponse {
	return &ErrorResponse{
		Err:            err,
		HTTPStatusCode: http.StatusUnprocessableEntity,
		StatusText:     "Error rendering response",
		ErrorText:      err.Error(),
	}
}

// ErrorNotFound return 404 with given error text
func ErrorNotFound(errorText string) *ErrorResponse {
	return &ErrorResponse{
		HTTPStatusCode: http.StatusNotFound,
		StatusText:     "Resource not found",
		ErrorText:      errorText,
	}
}

// ErrorForbidden return 403 with given error text
func ErrorForbidden(errorText string) *ErrorResponse {
	return &ErrorResponse{
		HTTPStatusCode: http.StatusForbidden,
		StatusText:     "Forbidden",
		ErrorText:      errorText,
	}
}

// ErrorRemoteServerUnavailable return 503 when remote trigger check failed
func ErrorRemoteServerUnavailable(err error) *ErrorResponse {
	return &ErrorResponse{
		Err:            err,
		HTTPStatusCode: http.StatusServiceUnavailable,
		StatusText:     "Remote server unavailable.",
		ErrorText:      fmt.Sprintf("Remote server error, please contact administrator. Raw error: %s", err.Error()),
	}
}

// ErrNotFound is default router page not found
var ErrNotFound = &ErrorResponse{HTTPStatusCode: http.StatusNotFound, StatusText: "Page not found."}

// ErrMethodNotAllowed is default 405 router method not allowed
var ErrMethodNotAllowed = &ErrorResponse{HTTPStatusCode: http.StatusMethodNotAllowed, StatusText: "Method not allowed."}
