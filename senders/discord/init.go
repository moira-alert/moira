package discord

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/worker"
)

const (
	messenger       = "discord"
	discordLockName = "moira-discord-users:moira-bot-host"
	discordLockTTL  = 30 * time.Second
	workerName      = "DiscordBot"
)

// Sender implements moira sender interface for discord
type Sender struct {
	DataBase  moira.Database
	logger    moira.Logger
	location  *time.Location
	session   *discordgo.Session
	frontURI  string
	botUserID string
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

	handleMsg := func(s *discordgo.Session, m *discordgo.MessageCreate) {
		channel, err := s.Channel(m.ChannelID)
		if err != nil {
			sender.logger.Error().
				Error(err).
				Msg("error while getting the channel details")
		}

		msg, err := sender.getResponse(m, channel)
		if err != nil {
			sender.logger.Error().
				Error(err).
				Msg("failed to handle incoming message")
		}
		s.ChannelMessageSend(m.ChannelID, msg) //nolint
	}
	sender.session.AddHandler(handleMsg)

	go sender.runBot()
	return nil
}

func (sender *Sender) runBot() {
	workerAction := func(stop <-chan struct{}) error {
		err := sender.session.Open()
		if err != nil {
			sender.logger.Error().
				Error(err).
				Msg("error creating a connection to discord")
			return nil
		}
		sender.botUserID = sender.session.State.User.ID
		defer sender.session.Close()
		<-stop
		return nil
	}

	worker.NewWorker(
		workerName,
		sender.logger,
		sender.DataBase.NewLock(discordLockName, discordLockTTL),
		workerAction,
	).Run(nil)
}
