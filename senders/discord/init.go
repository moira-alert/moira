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

// Sender implements moira sender interface for discord
type Sender struct {
	logger    moira.Logger
	location  *time.Location
	session   *discordgo.Session
	username  string
	avatarURL string
	frontURI  string
}

// Init reads the yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var err error
	token := senderSettings["token"]
	if token == "" {
		return fmt.Errorf("Cannot read the discord token from the config")
	}
	sender.session, err = discordgo.New("Bot " + token)
	if err != nil {
		return fmt.Errorf("Error creating discord session: %s", err)
	}
	sender.logger = logger
	sender.username = senderSettings["username"]
	sender.avatarURL = senderSettings["avatarURL"]
	sender.frontURI = senderSettings["front_uri"]
	sender.location = location

	go sender.runBot()
	return nil
}

func (sender *Sender) runBot() {
	sender.session.AddHandler(handleMsg)
	// Open a websocket connection to Discord and begin listening.
	err := sender.session.Open()
	if err != nil {
		sender.logger.Fatal("Error opening connection to discord: %s", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	sender.session.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func handleMsg(s *discordgo.Session, m *discordgo.MessageCreate) {
	return
}
