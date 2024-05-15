package discord

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/worker"
)

const (
	messenger      = "discord"
	discordLockKey = "moira-discord-users:moira-bot-host:"
	discordLockTTL = 30 * time.Second
	workerName     = "DiscordBot"
)

// Structure that represents the Discord configuration in the YAML file.
type config struct {
	ContactType string `mapstructure:"contact_type"`
	Token       string `mapstructure:"token"`
	FrontURI    string `mapstructure:"front_uri"`
}

// Sender implements moira sender interface for discord.
type Sender struct {
	DataBase  moira.Database
	logger    moira.Logger
	location  *time.Location
	session   *discordgo.Session
	frontURI  string
	botUserID string
}

// Init reads the yaml config.
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var cfg config
	err := mapstructure.Decode(senderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to discord config: %w", err)
	}

	if cfg.Token == "" {
		return fmt.Errorf("cannot read the discord token from the config")
	}
	sender.session, err = discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return fmt.Errorf("error creating discord session: %w", err)
	}

	sender.logger = logger
	sender.frontURI = cfg.FrontURI
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

	sender.runBot(cfg.ContactType)

	return nil
}

func (sender *Sender) runBot(contactType string) {
	err := sender.session.Open()
	if err != nil {
		sender.logger.Error().
			Error(err).
			Msg("error creating a connection to discord")
		return
	}
	sender.botUserID = sender.session.State.User.ID

	workerAction := func(stop <-chan struct{}) error {
		defer sender.session.Close()
		<-stop
		return nil
	}

	worker.NewWorker(
		workerName,
		sender.logger,
		sender.DataBase.NewLock(discordLock(contactType), discordLockTTL),
		workerAction,
	).Run(nil)
}

func discordLock(name string) string {
	return discordLockKey + name
}
