package discord

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/moira-alert/moira"
)

// Sender implements moira sender interface for discord
type Sender struct {
	logger   moira.Logger
	location *time.Location
	session  *discordgo.Session
	frontURI string
}

// Init reads the yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var err error
	token := senderSettings["token"]
	if token == "" {
		return fmt.Errorf("cannot read the discord token from the config")
	}
	sender.session, err = discordgo.New("Bot " + token)
	if err != nil {
		return fmt.Errorf("error creating discord session: %s", err)
	}
	sender.logger = logger
	sender.frontURI = senderSettings["front_uri"]
	sender.location = location

	return nil
}
