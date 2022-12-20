package zerolog_adapter

import (
	"github.com/moira-alert/moira/logging"
	"github.com/rs/zerolog"
)

type EventBuilder struct {
	event *zerolog.Event
}

func (e EventBuilder) Msg(msg string) {
	if e.event != nil {
		e.event.Timestamp().Msg(msg)
	}
}

func (e EventBuilder) String(key, value string) logging.EventBuilder {
	if e.event != nil {
		e.event.Str(key, value)
	}
	return e
}

func (e EventBuilder) Error(err error) logging.EventBuilder {
	if e.event != nil {
		e.event.Err(err)
	}
	return e
}

func (e EventBuilder) Int(key string, value int) logging.EventBuilder {
	if e.event != nil {
		e.event.Int(key, value)
	}
	return e
}

func (e EventBuilder) Int64(key string, value int64) logging.EventBuilder {
	if e.event != nil {
		e.event.Int64(key, value)
	}
	return e
}

func (e EventBuilder) Interface(key string, value interface{}) logging.EventBuilder {
	if e.event != nil {
		e.event.Interface(key, value)
	}
	return e
}

func (e EventBuilder) Fields(fields map[string]interface{}) logging.EventBuilder {
	if e.event != nil {
		e.event.Fields(fields)
	}
	return e
}
