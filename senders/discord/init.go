package main

import (
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
	sender.session = discord.New()
	sender.logger = logger
	sender.frontURI = senderSettings["front_uri"]
	sender.location = location
	return nil
}
