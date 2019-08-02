package opsgenie

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/opsgenie/opsgenie-go-sdk-v2/alert"
	"github.com/opsgenie/opsgenie-go-sdk-v2/client"
)

// Sender implements the Sender interface for opsgenie
type Sender struct {
	apiKey   string
	client   *alert.Client
	logger   moira.Logger
	location *time.Location

	frontURI string
}

// Init initializes the opsgenie sender
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var ok bool
	if sender.apiKey, ok = senderSettings["api_key"]; !ok {
		return fmt.Errorf("cannot read the api_key from the sender settings")
	}
	var err error
	sender.client, err = alert.NewClient(&client.Config{
		ApiKey: sender.apiKey,
	})
	if err != nil {
		return fmt.Errorf("error while creating opsgenie client: %s", err)
	}
	return nil
}
