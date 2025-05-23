package middleware

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/logging"
)

type apiLoggerEntry struct {
	logger  moira.Logger
	request *http.Request
	msg     string
}

// GetLoggerEntry gets logger entry with configured logger.
func GetLoggerEntry(request *http.Request) moira.Logger {
	apiLoggerEntry := request.Context().Value(middleware.LogEntryCtxKey).(*apiLoggerEntry)
	return apiLoggerEntry.logger
}

// WithLogEntry sets to context configured logger entry.
func WithLogEntry(r *http.Request, entry *apiLoggerEntry) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), middleware.LogEntryCtxKey, entry))
}

// RequestLogger is overload method of go-chi.middleware RequestLogger with custom response logging.
func RequestLogger(logger moira.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(writer http.ResponseWriter, request *http.Request) {
			entry := NewLogEntry(logger, request)
			wrapWriter := middleware.NewWrapResponseWriter(&responseWriterWithBody{ResponseWriter: writer}, request.ProtoMajor)

			t1 := time.Now()

			defer func() {
				rvr := recover()

				entry.fillMsg(request)

				if rvr != nil {
					render.Render(wrapWriter, request, api.ErrorInternalServer(fmt.Errorf("internal Server Error"))) //nolint
					entry.writePanic(wrapWriter.Status(), wrapWriter.BytesWritten(), time.Since(t1), rvr, debug.Stack())
				} else {
					entry.write(wrapWriter.Status(), wrapWriter.BytesWritten(), time.Since(t1), wrapWriter.Unwrap())
				}
			}()

			next.ServeHTTP(wrapWriter, WithLogEntry(request, entry))
		}

		return http.HandlerFunc(fn)
	}
}

func getErrorResponseIfItHas(writer http.ResponseWriter) *api.ErrorResponse {
	writerWithBody := writer.(*responseWriterWithBody)
	errResp := &api.ErrorResponse{}

	if err := json.NewDecoder(&writerWithBody.body).Decode(errResp); err != nil {
		return &api.ErrorResponse{
			HTTPStatusCode: http.StatusInternalServerError,
			Err:            err,
			ErrorText:      err.Error(),
		}
	}

	return errResp
}

// NewLogEntry is a function that creates an api logger entry.
func NewLogEntry(logger moira.Logger, request *http.Request) *apiLoggerEntry {
	entry := &apiLoggerEntry{
		logger:  logger.Clone(),
		request: request,
		msg:     "",
	}

	scheme := getScheme(request.TLS)
	uri := fmt.Sprintf("%s://%s%s", scheme, request.Host, request.RequestURI)
	log := entry.logger
	log.String("context", scheme)
	log.String("http.method", request.Method)
	log.String("http.uri", uri)
	log.String("http.protocol", request.Proto)
	log.String("http.remote_addr", request.RemoteAddr)
	log.String("username", GetLogin(request))

	return entry
}

func getScheme(tls *tls.ConnectionState) string {
	if tls != nil {
		return "https"
	}

	return "http"
}

func (entry *apiLoggerEntry) fillMsg(request *http.Request) {
	pattern := chi.RouteContext(request.Context()).RoutePattern()
	if pattern == "" {
		return
	}

	uri := fmt.Sprintf("%s://%s%s", getScheme(request.TLS), request.Host, pattern)
	entry.msg = fmt.Sprintf("%s %s %s", request.Method, uri, request.Proto)
}

func (entry *apiLoggerEntry) write(status, bytes int, elapsed time.Duration, response http.ResponseWriter) {
	var event logging.EventBuilder

	if status == 0 {
		status = http.StatusOK
	}

	if status >= http.StatusInternalServerError {
		event = entry.logger.Error()

		errorResponse := getErrorResponseIfItHas(response)
		if errorResponse != nil {
			event.Error(errorResponse.Err)
			event.String("error_text", errorResponse.ErrorText)
		}
	} else {
		event = entry.logger.Info()
	}

	event.Int("http.status", status).
		Int("http.content_length", bytes).
		Int64("elapsed_time_ms", elapsed.Milliseconds()).
		String("elapsed_time", elapsed.String()).
		Msg(entry.msg)
}

func (entry *apiLoggerEntry) writePanic(status, bytes int, elapsed time.Duration, v interface{}, stack []byte) {
	entry.logger.Error().
		Int("http_status", status).
		Int("http_content_length", bytes).
		Int("elapsed_time_ms", int(elapsed.Milliseconds())).
		Interface("recovered_err", v).
		String(moira.LogFieldNameStackTrace, string(stack)).
		Msg(fmt.Sprintf("%s: panic", entry.msg))
}

type responseWriterWithBody struct {
	http.ResponseWriter
	body bytes.Buffer
}

func (w *responseWriterWithBody) Write(buf []byte) (int, error) {
	n, err := w.ResponseWriter.Write(buf)
	_, err2 := w.body.Write(buf[:n])

	if err == nil {
		err = err2
	}

	return n, err
}
