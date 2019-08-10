package pagerduty

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"
)

// Sender implements moira sender interface for pagerduty
type Sender struct {
	ImageStores          map[string]moira.ImageStore
	imageStoreID         string
	imageStore           moira.ImageStore
	imageStoreConfigured bool
	logger               moira.Logger
	frontURI             string
	location             *time.Location
}

// Init loads yaml config, configures the pagerduty client
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	sender.frontURI = senderSettings["front_uri"]

	sender.imageStoreID, sender.imageStore, sender.imageStoreConfigured =
		senders.ReadImageStoreConfig(senderSettings, sender.ImageStores, logger)

	sender.logger = logger
	sender.location = location
	return nil
}
