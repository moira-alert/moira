package pagerduty

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
)

// Sender implements moira sender interface for pagerduty
type Sender struct {
	ImageStores  map[string]moira.ImageStore
	imageStoreID string
	logger       moira.Logger
	frontURI     string
	location     *time.Location
}

// Init loads yaml config, configures the pagerduty client
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	sender.frontURI = senderSettings["front_uri"]
	sender.imageStoreID = senderSettings["image_store"]
	if sender.imageStoreID == "" {
		return fmt.Errorf("cannot read image_store from the config")
	}
	sender.logger = logger
	sender.location = location
	return nil
}
