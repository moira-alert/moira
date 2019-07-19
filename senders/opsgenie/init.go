package opsgenie


import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/opsgenie/opsgenie-go-sdk-v2/alert"
)

// Sender implements the Sender interface for opsgenie
type Sender struct {
	apiKey string
	client *alert.Client
	logger   moira.Logger
	location *time.Location

	frontURI string
}

// Init initializes the opsgenie sender
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var ok bool
	sender.apiKey, ok = map["api_key"]; !ok{
		return fmt.Errorf("cannot read the api_key from the sender settings")
	}
	sender.client, err := alert.NewClient(&client.Config{
		ApiKey: sender.apiKey,
		logger: logger,
	})
}
