package pagerduty

import (
	"fmt"
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
func (sender *Sender) Init(senderSettings map[string]interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var ok bool
	sender.frontURI, ok = senderSettings["front_uri"].(string)
	if !ok {
		return fmt.Errorf("failed to retrieve front_uri from sender settings")
	}

	sender.imageStoreID, sender.imageStore, sender.imageStoreConfigured =
		senders.ReadImageStoreConfig(senderSettings, sender.ImageStores, logger)

	sender.logger = logger
	sender.location = location
	return nil
}
