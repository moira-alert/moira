package opsgenie

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier/senders"

	"github.com/opsgenie/opsgenie-go-sdk-v2/alert"
	"github.com/opsgenie/opsgenie-go-sdk-v2/client"
)

// Sender implements the Sender interface for Opsgenie.
// Use NewSender to create instance.
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

// NewSender creates Sender instance.
func NewSender(senderSettings map[string]string, logger moira.Logger, location *time.Location, imageStores map[string]moira.ImageStore) (*Sender, error) {
	sender := &Sender{
		ImageStores: imageStores,
	}

	var ok bool

	if sender.apiKey, ok = senderSettings["api_key"]; !ok {
		return nil, fmt.Errorf("cannot read the api_key from the sender settings")
	}

	sender.imageStoreID, sender.imageStore, sender.imageStoreConfigured =
		senders.ReadImageStoreConfig(senderSettings, sender.ImageStores, logger)

	var err error
	sender.client, err = alert.NewClient(&client.Config{
		ApiKey: sender.apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("error while creating opsgenie client: %s", err)
	}

	sender.frontURI = senderSettings["front_uri"]
	sender.logger = logger
	sender.location = location

	return sender, nil
}
