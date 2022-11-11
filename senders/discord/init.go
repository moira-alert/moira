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

// Sender implements moira sender interface for Discord.
// Use NewSender to create instance.
type Sender struct {
	DataBase  moira.Database
	logger    moira.Logger
	location  *time.Location
	session   *discordgo.Session
	frontURI  string
	botUserID string
}

// NewSender creates Sender instance.
func NewSender(senderSettings map[string]string, logger moira.Logger, location *time.Location, db moira.Database) (*Sender, error) {
	sender := &Sender{
		DataBase: db,
	}

	var err error
	token := senderSettings["token"]
	if token == "" {
		return nil, fmt.Errorf("cannot read the discord token from the config")
	}
	sender.session, err = discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("error creating discord session: %s", err)
	}
	sender.logger = logger
	sender.frontURI = senderSettings["front_uri"]
	sender.location = location

	handleMsg := func(s *discordgo.Session, m *discordgo.MessageCreate) {
		channel, err := s.Channel(m.ChannelID)
		if err != nil {
			sender.logger.Errorf("error while getting the channel details: %s", err)
		}

		msg, err := sender.getResponse(m, channel)
		if err != nil {
			sender.logger.Errorf("failed to handle incoming message: %s", err)
		}
		s.ChannelMessageSend(m.ChannelID, msg) //nolint
	}
	sender.session.AddHandler(handleMsg)

	go sender.runBot()

	return sender, nil
}

func (sender *Sender) runBot() {
	workerAction := func(stop <-chan struct{}) error {
		err := sender.session.Open()
		if err != nil {
			sender.logger.Errorf("error creating a connection to discord: %s", err)
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
