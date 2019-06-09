package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/moira-alert/moira"
)

// SendEvents implements pushover build and send message functionality
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {
	webhookParams := &discordgo.WebhookParams{}

	sender.logger.Debugf("Calling discord with message %s", webhookParams.Content)
	webhookID := ""
	webhookToken := ""
	err := sender.session.WebhookExecute(webhookID, webhookToken, false, webhookParams)
	if err != nil {
		return fmt.Errorf("failed to send %s event message to discord webhook %s : %s", trigger.ID, webhookID, err.Error())
	}
	return nil
}
