package logging

import (
	"github.com/moira-alert/moira"
	"github.com/rs/zerolog"
)

type EventBuilder struct {
	*zerolog.Event
}

func (e EventBuilder) Msg(msg string) {
	if e.Event != nil {
		e.Msg(msg)
	}
}

func (e EventBuilder) String(key, value string) moira.EventBuilder {
	if e.Event != nil {
		e.Str(key, value)
	}
	return e
}

func (e EventBuilder) Int(key string, value int) moira.EventBuilder {
	if e.Event != nil {
		e.Int(key, value)
	}
	return e
}

func (e EventBuilder) Int64(key string, value int64) moira.EventBuilder {
	if e.Event != nil {
		e.Int64(key, value)
	}
	return e
}

func (e EventBuilder) Fields(fields map[string]interface{}) moira.EventBuilder {
	if e.Event != nil {
		e.Fields(fields)
	}
	return e
}
