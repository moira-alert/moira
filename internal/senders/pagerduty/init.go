package pagerduty

import (
	"time"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/moira-alert/moira/internal/senders"
)

// Sender implements moira sender interface for pagerduty
type Sender struct {
	ImageStores          map[string]moira2.ImageStore
	imageStoreID         string
	imageStore           moira2.ImageStore
	imageStoreConfigured bool
	logger               moira2.Logger
	frontURI             string
	location             *time.Location
}

// Init loads yaml config, configures the pagerduty client
func (sender *Sender) Init(senderSettings map[string]string, logger moira2.Logger, location *time.Location, dateTimeFormat string) error {
	sender.frontURI = senderSettings["front_uri"]

	sender.imageStoreID, sender.imageStore, sender.imageStoreConfigured =
		senders.ReadImageStoreConfig(senderSettings, sender.ImageStores, logger)

	sender.logger = logger
	sender.location = location
	return nil
}
