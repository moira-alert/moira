package pagerduty

import (
	"fmt"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/moira-alert/moira"
)

// Sender implements moira sender interface for pagerduty
type Sender struct {
	DataBase  moira.Database
	logger    moira.Logger
	authToken string
	frontURI  string
	location  *time.Location
	client    *pagerduty.Client
}

// Init loads yaml config, configures the pagerduty client
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	authToken := senderSettings["auth_token"]
	if authToken == "" {
		return fmt.Errorf("can not read pagerduty auth_token from config")
	}

	sender.authToken = authToken
	sender.frontURI = senderSettings["front_uri"]
	sender.logger = logger
	sender.location = location
	var err error
	sender.client, err = pagerduty.NewClient(authToken)
	if err != nil {
		return err
	}

	return nil
}
