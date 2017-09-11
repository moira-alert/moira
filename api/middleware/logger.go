package middleware

import (
	"bytes"
	"fmt"
	"github.com/go-chi/chi/middleware"
	"github.com/moira-alert/moira"
	"net/http"
	"time"
)

// GetLoggerEntry gets logger entry with configured logger and current request
func GetLoggerEntry(request *http.Request) moira.Logger {
	return middleware.GetLogEntry(request).(*apiLoggerEntry).Logger
}

// Logger is custom api middleware logger realization use given moira.Logger interface and logs api requests
// Based on https:// github.com/go-chi/chi/blob/master/middleware/logger.go
func Logger(logger moira.Logger) func(next http.Handler) http.Handler {
	return middleware.RequestLogger(&apiLogger{logger})
}

type apiLogger struct {
	Logger moira.Logger
}

func (logger *apiLogger) NewLogEntry(request *http.Request) middleware.LogEntry {
	entry := &apiLoggerEntry{
		apiLogger: logger,
		request:   request,
		buf:       &bytes.Buffer{},
	}

	entry.buf.WriteString("\"")
	fmt.Fprintf(entry.buf, "%s ", request.Method)
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	fmt.Fprintf(entry.buf, "%s:// %s%s %s\" ", scheme, request.Host, request.RequestURI, request.Proto)
	entry.buf.WriteString("from ")
	entry.buf.WriteString(request.RemoteAddr)
	entry.buf.WriteString(" - ")
	return entry
}

type apiLoggerEntry struct {
	*apiLogger
	request *http.Request
	buf     *bytes.Buffer
}

func (entry *apiLoggerEntry) Write(status, bytes int, elapsed time.Duration) {
	fmt.Fprintf(entry.buf, "%03d", status)
	fmt.Fprintf(entry.buf, " %dB", bytes)
	entry.buf.WriteString(" in ")
	fmt.Fprintf(entry.buf, "%s", elapsed)
	entry.Logger.Info(entry.buf.String())
}

func (entry *apiLoggerEntry) Panic(v interface{}, stack []byte) {
	panicEntry := entry.NewLogEntry(entry.request).(*apiLoggerEntry)
	fmt.Fprintf(panicEntry.buf, "panic: %+v", v)
	entry.Logger.Info(panicEntry.buf.String())
	entry.Logger.Info(string(stack))
}
