package msgformat

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"
	"github.com/moira-alert/moira/senders/emoji_provider"
)

// UriFormatter is used for formatting uris, for example for Markdown use something like
// fmt.Sprintf("[%s](%s)", triggerName, triggerURI).
type UriFormatter func(triggerURI, triggerName string) string

// DescriptionFormatter is used to format trigger description to supported description.
type DescriptionFormatter func(trigger moira.TriggerData) string

// BoldFormatter makes str bold. For example in Markdown it should return **str**.
type BoldFormatter func(str string) string

// EventStringFormatter formats single event string.
type EventStringFormatter func(event moira.NotificationEvent, location *time.Location) string

// DescriptionCutter cuts the given description to fit max size.
type DescriptionCutter func(desc string, maxSize int) string

// TagsLimiter should prepare tags string in format like " [tag1][tag2][tag3]",
// but characters count should be less than or equal to maxSize.
type TagsLimiter func(tags []string, maxSize int) string

// highlightSyntaxFormatter formats message by using functions, emojis and some other highlight patterns.
type highlightSyntaxFormatter struct {
	// emojiGetter used in titles for better description.
	emojiGetter           emoji_provider.StateEmojiGetter
	frontURI              string
	location              *time.Location
	useEmoji              bool
	uriFormatter          UriFormatter
	descriptionFormatter  DescriptionFormatter
	descriptionCutter     DescriptionCutter
	boldFormatter         BoldFormatter
	eventsStringFormatter EventStringFormatter
	codeBlockStart        string
	codeBlockEnd          string
}

// NewHighlightSyntaxFormatter creates new highlightSyntaxFormatter with given arguments.
func NewHighlightSyntaxFormatter(
	emojiGetter emoji_provider.StateEmojiGetter,
	useEmoji bool,
	frontURI string,
	location *time.Location,
	uriFormatter UriFormatter,
	descriptionFormatter DescriptionFormatter,
	descriptionCutter DescriptionCutter,
	boldFormatter BoldFormatter,
	eventsStringFormatter EventStringFormatter,
	codeBlockStart string,
	codeBlockEnd string,
) MessageFormatter {
	return &highlightSyntaxFormatter{
		emojiGetter:           emojiGetter,
		frontURI:              frontURI,
		location:              location,
		useEmoji:              useEmoji,
		uriFormatter:          uriFormatter,
		descriptionFormatter:  descriptionFormatter,
		descriptionCutter:     descriptionCutter,
		boldFormatter:         boldFormatter,
		eventsStringFormatter: eventsStringFormatter,
		codeBlockStart:        codeBlockStart,
		codeBlockEnd:          codeBlockEnd,
	}
}

// Format formats message using given params and formatter functions.
func (formatter *highlightSyntaxFormatter) Format(params MessageFormatterParams) string {
	state := params.Events.GetCurrentState(params.Throttled)
	emoji := formatter.emojiGetter.GetStateEmoji(state)

	title := formatter.buildTitle(params.Events, params.Trigger, emoji, params.Throttled)
	titleLen := utf8.RuneCountInString(title) + len("\n")

	tagsStr := " " + params.Trigger.GetTags()
	tagsLen := utf8.RuneCountInString(tagsStr)

	desc := formatter.descriptionFormatter(params.Trigger)
	descLen := utf8.RuneCountInString(desc)

	eventsString := formatter.buildEventsString(params.Events, -1, params.Throttled)
	eventsStringLen := utf8.RuneCountInString(eventsString)

	charsLeftAfterTitle := params.MessageMaxChars - titleLen

	tagsNewLen, descNewLen, eventsNewLen := senders.CalculateMessageParts(charsLeftAfterTitle, tagsLen, descLen, eventsStringLen)
	if tagsNewLen != tagsLen {
		tagsStr = DefaultTagsLimiter(params.Trigger.Tags, tagsNewLen)
	}
	if descLen != descNewLen {
		desc = formatter.descriptionCutter(desc, descNewLen)
	}
	if eventsNewLen != eventsStringLen {
		eventsString = formatter.buildEventsString(params.Events, eventsNewLen, params.Throttled)
	}

	return title + tagsStr + "\n" + desc + eventsString
}

// buildTitle builds title string for alert (emoji, trigger state, trigger name with link).
func (formatter *highlightSyntaxFormatter) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData, emoji string, throttled bool) string {
	state := events.GetCurrentState(throttled)
	title := ""
	if formatter.useEmoji {
		title += emoji + " "
	}

	title += formatter.boldFormatter(string(state))
	triggerURI := trigger.GetTriggerURI(formatter.frontURI)
	if triggerURI != "" {
		title += fmt.Sprintf(" %s", formatter.uriFormatter(triggerURI, trigger.Name))
	} else if trigger.Name != "" {
		title += " " + trigger.Name
	}

	return title
}

// buildEventsString builds the string from moira events and limits it to charsForEvents.
// if charsForEvents is negative buildEventsString does not limit the events string.
func (formatter *highlightSyntaxFormatter) buildEventsString(events moira.NotificationEvents, charsForEvents int, throttled bool) string {
	charsForThrottleMsg := 0
	throttleMsg := fmt.Sprintf("\nPlease, %s to generate less events.", formatter.boldFormatter(ChangeTriggerRecommendation))
	if throttled {
		charsForThrottleMsg = utf8.RuneCountInString(throttleMsg)
	}
	charsLeftForEvents := charsForEvents - charsForThrottleMsg

	var eventsString string
	eventsString += formatter.codeBlockStart
	var tailString string

	eventsLenLimitReached := false
	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("\n%s", formatter.eventsStringFormatter(event, formatter.location))
		if msg := event.CreateMessage(formatter.location); len(msg) > 0 {
			line += fmt.Sprintf(". %s", msg)
		}

		tailString = fmt.Sprintf("\n...and %d more events.", len(events)-eventsPrinted)
		tailStringLen := utf8.RuneCountInString(formatter.codeBlockEnd) + len("\n") + utf8.RuneCountInString(tailString)
		if !(charsForEvents < 0) && (utf8.RuneCountInString(eventsString)+utf8.RuneCountInString(line) > charsLeftForEvents-tailStringLen) {
			eventsLenLimitReached = true
			break
		}

		eventsString += line
		eventsPrinted++
	}
	eventsString += "\n"
	eventsString += formatter.codeBlockEnd

	if eventsLenLimitReached {
		eventsString += tailString
	}

	if throttled {
		eventsString += throttleMsg
	}

	return eventsString
}
