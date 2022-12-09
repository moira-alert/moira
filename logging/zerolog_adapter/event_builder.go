package zerolog_dapter

import (
	"fmt"

	"github.com/moira-alert/moira/logging"
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

func (e EventBuilder) String(key, value string) logging.EventBuilder {
	if e.Event != nil {
		e.Event.Str(key, value)
	}
	return e
}

func (e EventBuilder) Error(err error) logging.EventBuilder {
	if e.Event != nil {
		e.Event.Str("error", err.Error())
	}
	return e
}

func (e EventBuilder) Int(key string, value int) logging.EventBuilder {
	if e.Event != nil {
		e.Event.Int(key, value)
	}
	return e
}

func (e EventBuilder) Int64(key string, value int64) logging.EventBuilder {
	if e.Event != nil {
		e.Event.Int64(key, value)
	}
	return e
}

func (e EventBuilder) Value(key string, value interface{}) logging.EventBuilder {
	return e.String(key, fmt.Sprintf("%v", value))
}

func (e EventBuilder) Fields(fields map[string]interface{}) logging.EventBuilder {
	if e.Event != nil {
		e.Event.Fields(fields)
	}
	return e
}
