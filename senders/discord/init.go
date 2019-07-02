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
	DataBase moira.Database
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

	handleMsg := func(s *discordgo.Session, m *discordgo.MessageCreate) {
		msg, err := sender.getResponse(s, m)
		if err != nil {
			sender.logger.Errorf("failed to handle incoming message: %s", err)
		}
		s.ChannelMessageSend(m.ChannelID, msg)
	}
	sender.session.AddHandler(handleMsg)

	go sender.runBot()
	return nil
}

func (sender *Sender) getResponse(s *discordgo.Session, m *discordgo.MessageCreate) (string, error) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return "", nil
	}

	// If the message is "!start" update the channel ID for the user/channel
	if m.Content == "!start" {
		channel, err := s.Channel(m.ChannelID)
		if err != nil {
			return "", fmt.Errorf("error while getting the channel details: %s", err)
		}
		switch channel.Type {
		case discordgo.ChannelTypeDM:
			err := sender.DataBase.SetUsernameID(messenger, "@"+m.Author.Username, channel.ID)
			if err != nil {
				return "", fmt.Errorf("error while setting the channel ID for user: %s", err)
			}
			msg := fmt.Sprintf("Okay, %s, your id is %s", m.Author.Username, channel.ID)
			return msg, nil
		case discordgo.ChannelTypeGuildText:
			err := sender.DataBase.SetUsernameID(messenger, channel.Name, channel.ID)
			if err != nil {
				return "", fmt.Errorf("error while setting the channel ID for text channel: %s", err)
			}
			msg := fmt.Sprintf("Hi, all!\nI will send alerts in this group (%s).", channel.Name)
			return msg, nil
		default:
			return "Unsupported channel type", nil
		}
	}

	return "", nil
}

func (sender *Sender) runBot() {
	workerAction := func(stop <-chan struct{}) error {
		err := sender.session.Open()
		if err != nil {
			sender.logger.Errorf("error creating a connection to discord: %s", err)
		}
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
