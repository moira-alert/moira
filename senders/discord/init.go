package discord

import (
	"time"

	"github.com/andersfylling/disgord"
	"github.com/moira-alert/moira"
)

// Sender implements moira sender interface for discord
type Sender struct {
	logger    moira.Logger
	location  *time.Location
	client    *disgord.Client
	username  string
	avatarURL string
	frontURI  string
}

// Init reads the yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var err error
	sender.client = disgord.New()
	if err != nil {
		return err
	}
	sender.logger = logger
	sender.username = senderSettings["username"]
	sender.avatarURL = senderSettings["avatarURL"]
	sender.frontURI = senderSettings["front_uri"]
	sender.location = location
	return nil
}
