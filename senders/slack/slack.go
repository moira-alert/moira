package slack

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/moira-alert/moira"

	"github.com/nlopes/slack"
)

var stateEmoji = map[string]string{
	"OK":        ":moira-state-ok:",
	"WARN":      ":moira-state-warn:",
	"ERROR":     ":moira-state-error:",
	"NODATA":    ":moira-state-nodata:",
	"EXCEPTION": ":moira-state-exception:",
	"TEST":      ":moira-state-test:",
}

// Sender implements moira sender interface via slack
type Sender struct {
	APIToken string
	FrontURI string
	useEmoji bool
	log      moira.Logger
	location *time.Location
}

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {

	sender.APIToken = senderSettings["api_token"]
	if sender.APIToken == "" {
		return fmt.Errorf("Can not read slack api_token from config")
	}
	var err error
	if useEmoji, ok := senderSettings["use_emoji"]; ok {
		sender.useEmoji, err = strconv.ParseBool(useEmoji)
		if err != nil {
			return fmt.Errorf("Can not read slack use_emoji parameter: %s", err.Error())
		}
	}
	sender.log = logger
	sender.FrontURI = senderSettings["front_uri"]
	sender.location = location
	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {

	api := slack.New(sender.APIToken)

	var message bytes.Buffer
	state := events.GetSubjectState()
	tags := trigger.GetTags()
	message.WriteString(fmt.Sprintf("*%s* %s <%s/trigger/%s|%s>\n %s \n```", state, tags, sender.FrontURI, events[0].TriggerID, trigger.Name, trigger.Desc))
	for _, event := range events {
		value := strconv.FormatFloat(moira.UseFloat64(event.Value), 'f', -1, 64)
		message.WriteString(fmt.Sprintf("\n%s: %s = %s (%s to %s)", time.Unix(event.Timestamp, 0).In(sender.location).Format("15:04"), event.Metric, value, event.OldState, event.State))
		if len(moira.UseString(event.Message)) > 0 {
			message.WriteString(fmt.Sprintf(". %s", moira.UseString(event.Message)))
		}
	}

	message.WriteString("```")

	if throttled {
		message.WriteString("\nPlease, *fix your system or tune this trigger* to generate less events.")
	}

	sender.log.Debugf("Calling slack with message body %s", message.String())

	params := slack.PostMessageParameters{
		Username:  "Moira",
		AsUser:    useDirectMessaging(contact.Value),
		IconEmoji: getStateEmoji(sender.useEmoji, state),
		Markdown:  true,
	}

	channelID, threadTimestamp, err := api.PostMessage(contact.Value, slack.MsgOptionText(message.String(),
		false), slack.MsgOptionPostMessageParameters(params))
	if err != nil {
		return fmt.Errorf("Failed to send %s event message to slack [%s]: %s", trigger.ID, contact.Value, err.Error())
	}

	if channelID != "" && len(plot) > 0 {
		reader := bytes.NewReader(plot)
		uploadParameters := slack.FileUploadParameters{
			Channels:        []string{channelID},
			ThreadTimestamp: threadTimestamp,
			Reader:          reader,
			Filetype:        "png",
			Filename:        fmt.Sprintf("%s.png", trigger.ID),
		}
		_, err := api.UploadFile(uploadParameters)
		if err != nil {
			sender.log.Errorf("Failed to send %s event plot to %s: %s", trigger.ID, contact.Value, err.Error())
		}
	}

	return nil
}

// getStateEmoji returns corresponding state emoji
func getStateEmoji(emojiEnabled bool, subjectState string) string {
	if emojiEnabled {
		if emoji, ok := stateEmoji[subjectState]; ok {
			return emoji
		}
	}
	return slack.DEFAULT_MESSAGE_ICON_EMOJI
}

// useDirectMessaging returns true if user contact is provided
func useDirectMessaging(contactValue string) bool {
	return len(contactValue) > 0 && contactValue[0:1] == "@"
}
