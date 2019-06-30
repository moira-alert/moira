package discord

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/moira-alert/moira"
)

const messenger = "discord"

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
		if err := sender.getResponse(s, m); err != nil {
			sender.logger.Errorf("failed to handle incoming message: %s", err)
		}
	}
	sender.session.AddHandler(handleMsg)

	go sender.runBot()
	return nil
}

func (sender *Sender) getResponse(s *discordgo.Session, m *discordgo.MessageCreate) error {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return nil
	}
	// If the message is "!start" update the channel ID for the user/channel
	if m.Content == "!start" {
		channel, err := s.Channel(m.ChannelID)
		if err != nil {
			return fmt.Errorf("error while getting the channel details: %s", err)
		}
		switch channel.Type {
		case discordgo.ChannelTypeDM:
			err := sender.DataBase.SetUsernameID(messenger, "@"+m.Author.Username, channel.ID)
			if err != nil {
				return fmt.Errorf("error while setting the channel ID for user: %s", err)
			}
			msg := fmt.Sprintf("Okay, %s, your id is %s", m.Author.Username, channel.ID)
			s.ChannelMessageSend(m.ChannelID, msg)
		case discordgo.ChannelTypeGuildText:
			uuid, _ := sender.DataBase.GetIDByUsername(messenger, channel.Name)
			err := sender.DataBase.SetUsernameID(messenger, channel.Name, channel.ID)
			if err != nil {
				return fmt.Errorf("error while setting the channel ID for text channel: %s", err)
			}
			if uuid == "" {
				msg := fmt.Sprintf("Hi, all!\nI will send alerts in this group (%s).", channel.Name)
				s.ChannelMessageSend(m.ChannelID, msg)
			}
		default:
			s.ChannelMessageSend(m.ChannelID, "Unsupported channel type")
		}
	}

	return nil
}

func (sender *Sender) runBot() {
	err := sender.session.Open()
	if err != nil {
		sender.logger.Errorf("error creating a connection to discord: %s", err)
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
	defer sender.session.Close()
}
