package discord

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"
)

const (
	messageMaxCharacters = 2000
)

var (
	mdHeaderRegex = regexp.MustCompile(`(?m)^\s*#{1,}\s*(?P<headertext>[^#\n]+)$`)
)

// SendEvents implements pushover build and send message functionality
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	data := &discordgo.MessageSend{}
	data.Content = sender.buildMessage(events, trigger, throttled)
	if len(plots) > 0 {
		data.File = sender.buildPlot(plots[0])
		data.Embed = &discordgo.MessageEmbed{
			Image: &discordgo.MessageEmbedImage{
				URL: "attachment://Plot.jpg",
			},
		}
	}
	sender.logger.Debugf("Calling discord with message %s", data.Content)
	channelID, err := sender.getChannelID(contact.Value)
	if err != nil {
		return fmt.Errorf("failed to get the channel ID: %s", err)
	}
	_, err = sender.session.ChannelMessageSendComplex(channelID, data)
	if err != nil {
		return fmt.Errorf("failed to send %s event message to discord bot : %s", trigger.ID, err.Error())
	}
	return nil
}

func (sender *Sender) getChannelID(username string) (string, error) {
	chid, err := sender.DataBase.GetIDByUsername(messenger, username)
	if err != nil {
		return "", fmt.Errorf("failed to get channel ID: %s", err.Error())
	}
	return chid, nil
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	var buffer strings.Builder

	state := events.GetSubjectState()
	tags := trigger.GetTags()
	title := fmt.Sprintf("%s %s %s (%d)\n", state, trigger.Name, tags, len(events))
	titleLen := len([]rune(title))

	desc := sender.buildDescription(trigger)
	descLen := len([]rune(desc))

	eventsString := sender.buildEventsString(events, -1, throttled, trigger)
	eventsStringLen := len([]rune(eventsString))

	charsLeftAfterTitle := messageMaxCharacters - titleLen

	descNewLen, eventsNewLen := senders.CalculateMessagePartsLength(charsLeftAfterTitle, descLen, eventsStringLen)

	if descLen != descNewLen {
		desc = desc[:descNewLen] + "...\n"
	}
	if eventsNewLen != eventsStringLen {
		eventsString = sender.buildEventsString(events, eventsNewLen, throttled, trigger)
	}

	buffer.WriteString(title)
	buffer.WriteString(desc)
	buffer.WriteString(eventsString)
	return buffer.String()
}

func (sender *Sender) buildDescription(trigger moira.TriggerData) string {
	desc := trigger.Desc
	if trigger.Desc != "" {
		// Replace MD headers (## header text) with **header text** that telegram supports
		desc = mdHeaderRegex.ReplaceAllString(trigger.Desc, "**$headertext**")
		desc += "\n"
	}
	return desc
}

// buildEventsString builds the string from moira events and limits it to charsForEvents.
// if n is negative buildEventsString does not limit the events string
func (sender *Sender) buildEventsString(events moira.NotificationEvents, charsForEvents int, throttled bool, trigger moira.TriggerData) string {
	charsForThrottleMsg := 0
	throttleMsg := "\nPlease, fix your system or tune this trigger to generate less events."
	if throttled {
		charsForThrottleMsg = len([]rune(throttleMsg))
	}

	var urlString string
	url := trigger.GetTriggerURI(sender.frontURI)
	if url != "" {
		urlString = fmt.Sprintf("\n\n%s\n", url)
	}
	charsLeftForEvents := charsForEvents - len([]rune(urlString)) - charsForThrottleMsg

	var eventsString string
	var tailString string
	eventsLenLimitReached := false
	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricsValues(), event.OldState, event.State)
		if msg := event.CreateMessage(sender.location); len(msg) > 0 {
			line += fmt.Sprintf(". %s", msg)
		}
		tailString = fmt.Sprintf("\n\n...and %d more events.", len(events)-eventsPrinted)
		tailStringLen := len([]rune(tailString))
		if !(charsForEvents < 0) && (len([]rune(eventsString))+len([]rune(line)) > charsLeftForEvents-tailStringLen) {
			eventsLenLimitReached = true
			break
		}

		eventsString += line
		eventsPrinted++

	}

	if eventsLenLimitReached {
		eventsString += tailString
	}
	if url != "" {
		eventsString += urlString
	}
	if throttled {
		eventsString += throttleMsg
	}

	return eventsString
}

func (sender *Sender) buildPlot(plot []byte) *discordgo.File {
	return &discordgo.File{
		Name:        "Plot.jpg",
		ContentType: http.DetectContentType(plot),
		Reader:      bytes.NewReader(plot),
	}
}
