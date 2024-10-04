package opsgenie

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"
	"github.com/opsgenie/opsgenie-go-sdk-v2/alert"
	"github.com/opsgenie/opsgenie-go-sdk-v2/client"
)

// Structure that represents the OpsGenie configuration in the YAML file.
type config struct {
	APIKey string `mapstructure:"api_key" validate:"required"`
}

// Sender implements the Sender interface for opsgenie.
type Sender struct {
	apiKey               string
	client               *alert.Client
	logger               moira.Logger
	location             *time.Location
	ImageStores          map[string]moira.ImageStore
	imageStoreID         string
	imageStore           moira.ImageStore
	imageStoreConfigured bool
}

// Init initializes the opsgenie sender.
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var cfg config

	err := mapstructure.Decode(senderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to opsgenie config: %w", err)
	}

	if err = moira.ValidateConfig(cfg); err != nil {
		return fmt.Errorf("opsgenie config validation error: %w", err)
	}

	sender.apiKey = cfg.APIKey
	sender.imageStoreID, sender.imageStore, sender.imageStoreConfigured = senders.ReadImageStoreConfig(senderSettings, sender.ImageStores, logger)

	sender.client, err = alert.NewClient(&client.Config{
		ApiKey: sender.apiKey,
	})
	if err != nil {
		return fmt.Errorf("error while creating opsgenie client: %w", err)
	}

	sender.logger = logger
	sender.location = location
	return nil
}
