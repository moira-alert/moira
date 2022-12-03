package logging

import (
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/rs/zerolog"
)

type EventBuilder struct {
	*zerolog.Event
}

func (e EventBuilder) Msg(msg string) {
	if e.Event != nil {
		e.Event.Msg(msg)
	}
}

func (e EventBuilder) String(key, value string) moira.EventBuilder {
	if e.Event != nil {
		e.Event.Str(key, value)
	}
	return e
}

func (e EventBuilder) Error(err error) moira.EventBuilder {
	if e.Event != nil {
		e.Event.Str("error", err.Error())
	}
	return e
}

func (e EventBuilder) Int(key string, value int) moira.EventBuilder {
	if e.Event != nil {
		e.Event.Int(key, value)
	}
	return e
}

func (e EventBuilder) Int64(key string, value int64) moira.EventBuilder {
	if e.Event != nil {
		e.Event.Int64(key, value)
	}
	return e
}

func (e EventBuilder) Value(key string, value interface{}) moira.EventBuilder {
	return e.String(key, fmt.Sprintf("%v", value))
}

func (e EventBuilder) Fields(fields map[string]interface{}) moira.EventBuilder {
	if e.Event != nil {
		e.Event.Fields(fields)
	}
	return e
}
