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
	messenger       = "discord"
	discordLockName = "moira-discord-users:moira-bot-host:"
	discordLockTTL  = 30 * time.Second
	workerName      = "DiscordBot"
)

// Structure that represents the Discord configuration in the YAML file
type config struct {
	Type     string `mapsctructure:"type"`
	Name     string `mapstructure:"name"`
	Token    string `mapstructure:"token"`
	FrontURI string `mapstructure:"front_uri"`
}

// Sender implements moira sender interface for discord
type Sender struct {
	clients map[string]*discordClient
}

type discordClient struct {
	frontURI  string
	botUserID string
	dataBase  moira.Database
	location  *time.Location
	session   *discordgo.Session
	logger    moira.Logger
}

// Init reads the yaml config
func (sender *Sender) Init(opts moira.InitOptions) error {
	var cfg config
	err := mapstructure.Decode(opts.SenderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to discord config: %w", err)
	}

	if cfg.Token == "" {
		return fmt.Errorf("cannot read the discord token from the config")
	}

	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return fmt.Errorf("error creating discord session: %s", err)
	}

	client := &discordClient{
		session:  session,
		logger:   opts.Logger,
		frontURI: cfg.FrontURI,
		location: opts.Location,
		dataBase: opts.Database,
	}

	handleMsg := func(s *discordgo.Session, m *discordgo.MessageCreate) {
		channel, err := s.Channel(m.ChannelID)
		if err != nil {
			client.logger.Error().
				Error(err).
				Msg("error while getting the channel details")
		}

		msg, err := client.getResponse(m, channel)
		if err != nil {
			client.logger.Error().
				Error(err).
				Msg("failed to handle incoming message")
		}
		s.ChannelMessageSend(m.ChannelID, msg) //nolint
	}
	session.AddHandler(handleMsg)

	if sender.clients == nil {
		sender.clients = make(map[string]*discordClient)
	}

	var senderIdent string
	if cfg.Name != "" {
		senderIdent = cfg.Name
	} else {
		senderIdent = cfg.Type
	}

	sender.clients[senderIdent] = client

	go client.runBot(senderIdent)

	return nil
}

func (client *discordClient) runBot(senderIdent string) {
	workerAction := func(stop <-chan struct{}) error {
		err := client.session.Open()
		if err != nil {
			client.logger.Error().
				Error(err).
				Msg("error creating a connection to discord")
			return nil
		}
		client.botUserID = client.session.State.User.ID
		defer client.session.Close()
		<-stop
		return nil
	}

	worker.NewWorker(
		workerName,
		client.logger,
		client.dataBase.NewLock(discordLockName+senderIdent, discordLockTTL),
		workerAction,
	).Run(nil)
}
