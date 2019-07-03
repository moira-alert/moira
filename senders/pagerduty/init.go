package pagerduty

import (
	"time"

	"github.com/moira-alert/moira"
)

// Sender implements moira sender interface for pagerduty
type Sender struct {
	ImageStore moira.ImageStore
	logger     moira.Logger
	frontURI   string
	location   *time.Location
}

// Init loads yaml config, configures the pagerduty client
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	sender.frontURI = senderSettings["front_uri"]
	sender.logger = logger
	sender.location = location
	return nil
}
