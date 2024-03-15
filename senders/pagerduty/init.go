package pagerduty

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"
)

// Structure that represents the PagerDuty configuration in the YAML file.
type config struct {
	FrontURI string `mapstructure:"front_uri"`
}

// Sender implements moira sender interface for pagerduty.
type Sender struct {
	ImageStores          map[string]moira.ImageStore
	imageStoreID         string
	imageStore           moira.ImageStore
	imageStoreConfigured bool
	logger               moira.Logger
	frontURI             string
	location             *time.Location
}

// Init loads yaml config, configures the pagerduty client.
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var cfg config
	err := mapstructure.Decode(senderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to pagerduty config: %w", err)
	}

	sender.frontURI = cfg.FrontURI

	sender.imageStoreID, sender.imageStore, sender.imageStoreConfigured =
		senders.ReadImageStoreConfig(senderSettings, sender.ImageStores, logger)

	sender.logger = logger
	sender.location = location
	return nil
}
