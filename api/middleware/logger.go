package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
)

type apiLoggerEntry struct {
	logger  moira.Logger
	request *http.Request
	buf     *bytes.Buffer
}

// GetLoggerEntry gets logger entry with configured logger
func GetLoggerEntry(request *http.Request) moira.Logger {
	return request.Context().Value(middleware.LogEntryCtxKey).(*apiLoggerEntry).logger
}

// WithLogEntry sets to context configured logger entry
func WithLogEntry(r *http.Request, entry *apiLoggerEntry) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), middleware.LogEntryCtxKey, entry))
}

// RequestLogger is overload method of go-chi.middleware RequestLogger with custom response logging
func RequestLogger(logger moira.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(writer http.ResponseWriter, request *http.Request) {
			entry := newLogEntry(logger, request)
			wrapWriter := middleware.NewWrapResponseWriter(&responseWriterWithBody{ResponseWriter: writer}, request.ProtoMajor)

			t1 := time.Now()
			defer func() {
				if rvr := recover(); rvr != nil {
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
	var errResp = &api.ErrorResponse{}
	json.NewDecoder(&writerWithBody.body).Decode(errResp) //nolint
	return errResp
}

func newLogEntry(logger moira.Logger, request *http.Request) *apiLoggerEntry {
	entry := &apiLoggerEntry{
		logger:  logger.Clone(),
		request: request,
		buf:     &bytes.Buffer{},
	}

	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	userName := GetLogin(request)
	if userName == "" {
		userName = "anonymous"
	}
	uri := fmt.Sprintf("%s://%s%s", scheme, request.Host, request.RequestURI)

	log := entry.logger
	log.String("context", "http")
	log.String("http.method", request.Method)
	log.String("http.uri", uri)
	log.String("http.protocol", request.Proto)
	log.String("http.remote_addr", request.RemoteAddr)
	log.String("username", userName)

	entry.buf.WriteString("\"")
	fmt.Fprintf(entry.buf, "%s ", request.Method)
	fmt.Fprintf(entry.buf, "%s %s\"", uri, request.Proto)
	entry.buf.WriteString(" from ")
	entry.buf.WriteString(request.RemoteAddr)
	entry.buf.WriteString(" by ")
	entry.buf.WriteString(userName)
	entry.buf.WriteString(" - ")

	return entry
}

func (entry *apiLoggerEntry) write(status, bytes int, elapsed time.Duration, response http.ResponseWriter) {
	if status == 0 {
		status = http.StatusOK
	}
	log := entry.logger
	log.Int("http.http_status", status)
	log.Int("http.content_length", bytes)
	log.Int64("elapsed_time_ms", elapsed.Milliseconds())

	fmt.Fprintf(entry.buf, "%03d", status)
	fmt.Fprintf(entry.buf, " %dB", bytes)
	entry.buf.WriteString(" in ")
	fmt.Fprintf(entry.buf, "%s", elapsed)
	if status >= http.StatusInternalServerError {
		errorResponse := getErrorResponseIfItHas(response)
		if errorResponse != nil {
			fmt.Fprintf(entry.buf, " - Error : %s", errorResponse.ErrorText)
		}
		log.Error(entry.buf.String())
	} else {
		log.Info(entry.buf.String())
	}
}

func (entry *apiLoggerEntry) writePanic(status, bytes int, elapsed time.Duration, v interface{}, stack []byte) {
	log := entry.logger
	log.Int("http.http_status", status)
	log.Int("http.content_length", bytes)
	log.Int("elapsed_time_ms", int(elapsed.Milliseconds()))

	fmt.Fprintf(entry.buf, "%03d", status)
	fmt.Fprintf(entry.buf, " %dB", bytes)
	entry.buf.WriteString(" in ")
	fmt.Fprintf(entry.buf, "%s", elapsed)
	fmt.Fprintf(entry.buf, " - Panic: %+v", v)
	entry.buf.WriteString("\n")
	entry.buf.WriteString(string(stack))
	log.Error(entry.buf.String())
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
