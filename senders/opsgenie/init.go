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

// Structure that represents the OpsGenie configuration in the YAML file
type OpsGenie struct {
	APIKey   string `mapstructure:"api_key"`
	FrontURI string `mapstructure:"front_uri"`
}

// Sender implements the Sender interface for opsgenie
type Sender struct {
	apiKey               string
	client               *alert.Client
	logger               moira.Logger
	location             *time.Location
	ImageStores          map[string]moira.ImageStore
	imageStoreID         string
	imageStore           moira.ImageStore
	imageStoreConfigured bool
	frontURI             string
}

// Init initializes the opsgenie sender
func (sender *Sender) Init(senderSettings map[string]interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var opsgenie OpsGenie

	err := mapstructure.Decode(senderSettings, &opsgenie)
	if err != nil {
		return fmt.Errorf("decoding error from yaml file to opsgenie structure: %s", err)
	}

	sender.apiKey = opsgenie.APIKey
	if sender.apiKey == "" {
		return fmt.Errorf("cannot read the api_key from the sender settings")
	}

	sender.imageStoreID, sender.imageStore, sender.imageStoreConfigured =
		senders.ReadImageStoreConfig(senderSettings, sender.ImageStores, logger)

	sender.client, err = alert.NewClient(&client.Config{
		ApiKey: sender.apiKey,
	})
	if err != nil {
		return fmt.Errorf("error while creating opsgenie client: %s", err)
	}

	sender.frontURI = opsgenie.FrontURI
	sender.logger = logger
	sender.location = location
	return nil
}
