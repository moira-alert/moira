package opsgenie

import (
	"fmt"
	"time"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/moira-alert/moira/internal/senders"
	"github.com/opsgenie/opsgenie-go-sdk-v2/alert"
	"github.com/opsgenie/opsgenie-go-sdk-v2/client"
)

// Sender implements the Sender interface for opsgenie
type Sender struct {
	apiKey               string
	client               *alert.Client
	logger               moira2.Logger
	location             *time.Location
	ImageStores          map[string]moira2.ImageStore
	imageStoreID         string
	imageStore           moira2.ImageStore
	imageStoreConfigured bool
	frontURI             string
}

// Init initializes the opsgenie sender
func (sender *Sender) Init(senderSettings map[string]string, logger moira2.Logger, location *time.Location, dateTimeFormat string) error {
	var ok bool

	if sender.apiKey, ok = senderSettings["api_key"]; !ok {
		return fmt.Errorf("cannot read the api_key from the sender settings")
	}

	sender.imageStoreID, sender.imageStore, sender.imageStoreConfigured =
		senders.ReadImageStoreConfig(senderSettings, sender.ImageStores, logger)

	var err error
	sender.client, err = alert.NewClient(&client.Config{
		ApiKey: sender.apiKey,
	})
	if err != nil {
		return fmt.Errorf("error while creating opsgenie client: %s", err)
	}

	sender.frontURI = senderSettings["front_uri"]
	sender.logger = logger
	sender.location = location
	return nil
}
