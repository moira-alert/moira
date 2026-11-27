package dto

import (
	"errors"
	"net/http"
)

const (
	// DescriptionEventsReal indicates that the preview used existing trigger events.
	DescriptionEventsReal = "real"
	// DescriptionEventsDemo indicates that the preview used demo trigger events.
	DescriptionEventsDemo = "demo"
)

// TriggerDescriptionPreviewRequest is a structure that represents request body for previewing trigger description.
type TriggerDescriptionPreviewRequest struct {
	TriggerID   string  `json:"trigger_id,omitempty"`
	TriggerName string  `json:"name"`
	Description *string `json:"description" binding:"required"`
}

func (preview *TriggerDescriptionPreviewRequest) Bind(_ *http.Request) error {
	if preview.Description == nil {
		return errors.New("description must be provided as a string")
	}

	return nil
}

type TriggerDescriptionPreview struct {
	Description  string `json:"description"`
	EventsSource string `json:"events_source" enums:"real,demo"`
}

func (*TriggerDescriptionPreview) Render(http.ResponseWriter, *http.Request) error {
	return nil
}
