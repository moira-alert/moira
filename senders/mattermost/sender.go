package mattermost

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/moira-alert/moira/senders/message_builder"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/moira-alert/moira"
)

// Sender posts messages to Mattermost chat.
// It implements moira.Sender.
// You must call Init method before SendEvents method.
type Sender struct {
	*message_builder.MessageBuilder
	client Client
}

const messageMaxCharacters = 4000

// Init configures Sender.
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, _ string) error {
	url := senderSettings["url"]
	if url == "" {
		return fmt.Errorf("can not read Mattermost url from config")
	}
	client := model.NewAPIv4Client(url)

	insecureTLS, err := strconv.ParseBool(senderSettings["insecure_tls"])
	if err != nil {
		return fmt.Errorf("can not parse insecure_tls: %v", err)
	}
	client.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureTLS,
			},
		},
	}
	sender.client = client

	token := senderSettings["api_token"]
	if token == "" {
		return fmt.Errorf("can not read Mattermost api_token from config")
	}
	sender.client.SetToken(token)

	frontURI := senderSettings["front_uri"]
	if frontURI == "" {
		return fmt.Errorf("can not read Mattermost front_uri from config")
	}

	sender.MessageBuilder = message_builder.NewMessageBuilder(
		senderSettings["front_uri"],
		messageMaxCharacters,
		logger,
		location,
	)

	return nil
}

// SendEvents implements moira.Sender interface.
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, _ [][]byte, throttled bool) error {
	message := sender.BuildMessage(events, trigger, throttled)
	err := sender.sendMessage(message, contact.Value, trigger.ID)
	if err != nil {
		return err
	}

	return nil
}

func (sender *Sender) sendMessage(message string, contact string, triggerID string) error {
	if _, _, err := sender.client.CreatePost(&model.Post{ChannelId: contact, Message: message}); err != nil {
		return fmt.Errorf("failed to send %s event message to Mattermost [%s]: %s", triggerID, contact, err)
	}

	return nil
}
