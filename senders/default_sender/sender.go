package default_sender

import (
	"fmt"
	"strings"
	"time"

	slackdown "github.com/karriereat/blackfriday-slack"
	"github.com/moira-alert/moira/senders"
	"github.com/russross/blackfriday/v2"

	"github.com/moira-alert/moira"
)

// DefaultSender is struct for default sender.
type DefaultSender struct {
	Logger               moira.Logger
	frontURI             string
	messageMaxCharacters int
	location             *time.Location
}

// NewDefaultSender is construct for DefaultSender struct.
func NewDefaultSender(
	frontURI string,
	messageMaxCharacters int,
	logger moira.Logger,
	location *time.Location,
) *DefaultSender {
	return &DefaultSender{
		frontURI:             frontURI,
		messageMaxCharacters: messageMaxCharacters,
		Logger:               logger,
		location:             location,
	}
}

// BuildMessage makes message body this collapse events or description.
func (sender *DefaultSender) BuildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	var message strings.Builder

	title := sender.buildTitle(events, trigger)
	titleLen := len([]rune(title))

	desc := sender.buildDescription(trigger)
	descLen := len([]rune(desc))

	eventsString := sender.buildEventsString(events, -1, throttled)
	eventsStringLen := len([]rune(eventsString))

	charsLeftAfterTitle := sender.messageMaxCharacters - titleLen

	descNewLen, eventsNewLen := senders.CalculateMessagePartsLength(charsLeftAfterTitle, descLen, eventsStringLen)
	if descLen > descNewLen {
		lenPostfix := len([]rune("...\n"))
		desc = desc[:descNewLen-lenPostfix] + "...\n"
	}
	if eventsStringLen > eventsNewLen {
		eventsString = sender.buildEventsString(events, eventsNewLen, throttled)
	}

	message.WriteString(title)
	message.WriteString(desc)
	message.WriteString(eventsString)
	return message.String()
}

func (sender *DefaultSender) buildDescription(trigger moira.TriggerData) string {
	desc := trigger.Desc
	if trigger.Desc != "" {
		desc = string(blackfriday.Run([]byte(desc), blackfriday.WithRenderer(&slackdown.Renderer{})))
		desc += "\n"
	}
	return desc
}

func (sender *DefaultSender) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData) string {
	title := fmt.Sprintf("*%s*", events.GetSubjectState())
	triggerURI := trigger.GetTriggerURI(sender.frontURI)
	if triggerURI != "" {
		title += fmt.Sprintf(" <%s|%s>", triggerURI, trigger.Name)
	} else if trigger.Name != "" {
		title += " " + trigger.Name
	}

	tags := trigger.GetTags()
	if tags != "" {
		title += " " + tags
	}

	title += "\n"
	return title
}

// buildEventsString builds the string from moira events and limits it to charsForEvents.
// if n is negative buildEventsString does not limit the events string
func (sender *DefaultSender) buildEventsString(events moira.NotificationEvents, charsForEvents int, throttled bool) string {
	charsForThrottleMsg := 0
	throttleMsg := "\nPlease, *fix your system or tune this trigger* to generate less events."
	if throttled {
		charsForThrottleMsg = len([]rune(throttleMsg))
	}
	charsLeftForEvents := charsForEvents - charsForThrottleMsg

	var eventsString string
	eventsString += "```"
	var tailString string

	eventsLenLimitReached := false
	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricsValues(), event.OldState, event.State)
		if msg := event.CreateMessage(sender.location); len(msg) > 0 {
			line += fmt.Sprintf(". %s", msg)
		}

		tailString = fmt.Sprintf("\n...and %d more events.", len(events)-eventsPrinted)
		tailStringLen := len([]rune("```")) + len([]rune(tailString))
		if !(charsForEvents < 0) && (len([]rune(eventsString))+len([]rune(line)) > charsLeftForEvents-tailStringLen) {
			eventsLenLimitReached = true
			break
		}

		eventsString += line
		eventsPrinted++
	}
	eventsString += "```"

	if eventsLenLimitReached {
		eventsString += tailString
	}

	if throttled {
		eventsString += throttleMsg
	}

	return eventsString
}
