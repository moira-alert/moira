package pagerduty

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"
)

// Sender implements moira sender interface for PagerDuty.
// Use NewSender to create instance.
type Sender struct {
	ImageStores          map[string]moira.ImageStore
	imageStoreID         string
	imageStore           moira.ImageStore
	imageStoreConfigured bool
	logger               moira.Logger
	frontURI             string
	location             *time.Location
}

// NewSender creates Sender instance.
func NewSender(senderSettings map[string]string, logger moira.Logger, location *time.Location, imageStores map[string]moira.ImageStore) *Sender {
	sender := &Sender{
		ImageStores: imageStores,
	}

	sender.frontURI = senderSettings["front_uri"]

	sender.imageStoreID, sender.imageStore, sender.imageStoreConfigured =
		senders.ReadImageStoreConfig(senderSettings, sender.ImageStores, logger)

	sender.logger = logger
	sender.location = location

	return sender
}
