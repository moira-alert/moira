package dto

import "net/http"

type SentryConfig struct {
	DSN string `json:"dsn" example:"https://secret@sentry.host"`
}

func (SentryConfig) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
