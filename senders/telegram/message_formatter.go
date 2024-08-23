package telegram

import (
	"fmt"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"
	"github.com/moira-alert/moira/senders/emoji_provider"
	"github.com/moira-alert/moira/senders/msgformat"
	"github.com/russross/blackfriday/v2"
	"html"
	"regexp"
	"strings"
	"time"
)

const (
	codeBlockStart = "<blockquote expandable>"
	codeBlockEnd   = "</blockquote>"
)

type messageFormatter struct {
	// emojiGetter used in titles for better description.
	emojiGetter emoji_provider.StateEmojiGetter
	frontURI    string
	location    *time.Location
	useEmoji    bool
}

func NewTelegramMessageFormatter(
	emojiGetter emoji_provider.StateEmojiGetter,
	useEmoji bool,
	frontURI string,
	location *time.Location,
) msgformat.MessageFormatter {
	return &messageFormatter{
		emojiGetter: emojiGetter,
		frontURI:    frontURI,
		location:    location,
		useEmoji:    useEmoji,
	}
}

func (formatter *messageFormatter) Format(params msgformat.MessageFormatterParams) string {
	var message strings.Builder
	state := params.Events.GetCurrentState(params.Throttled)
	emoji := formatter.emojiGetter.GetStateEmoji(state)

	title := formatter.buildTitle(params.Events, params.Trigger, emoji, params.Throttled)
	titleLen := calcRunesCountWithoutHTML([]rune(title))

	desc := descriptionFormatter(params.Trigger)
	descLen := calcRunesCountWithoutHTML([]rune(desc))

	eventsString := formatter.buildEventsString(params.Events, -1, params.Throttled)
	eventsStringLen := calcRunesCountWithoutHTML([]rune(eventsString))

	descNewLen, eventsNewLen := senders.CalculateMessagePartsLength(params.MessageMaxChars-titleLen, descLen, eventsStringLen)
	if descLen != descNewLen {
		desc = descriptionCutter(desc, descNewLen)
	}
	if eventsNewLen != eventsStringLen {
		eventsString = formatter.buildEventsString(params.Events, eventsNewLen, params.Throttled)
	}

	message.WriteString(title)
	message.WriteString(desc)
	message.WriteString(eventsString)
	return message.String()
}

func calcRunesCountWithoutHTML(htmlText []rune) int {
	textLen := 0
	isTag := false

	for _, r := range htmlText {
		if r == '<' {
			isTag = true
			continue
		}

		if !isTag {
			textLen += 1
		}

		if r == '>' {
			isTag = false
		}
	}

	return textLen
}

func (formatter *messageFormatter) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData, emoji string, throttled bool) string {
	state := events.GetCurrentState(throttled)
	title := ""
	if formatter.useEmoji {
		title += emoji + " "
	}

	title += boldFormatter(string(state))
	triggerURI := trigger.GetTriggerURI(formatter.frontURI)
	if triggerURI != "" {
		title += fmt.Sprintf(" %s", uriFormatter(triggerURI, trigger.Name))
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
// if charsForEvents is negative buildEventsString does not limit the events string.
func (formatter *messageFormatter) buildEventsString(events moira.NotificationEvents, charsForEvents int, throttled bool) string {
	charsForThrottleMsg := 0
	throttleMsg := fmt.Sprintf("\nPlease, %s to generate less events.", boldFormatter(msgformat.ChangeTriggerRecommendation))
	if throttled {
		charsForThrottleMsg = calcRunesCountWithoutHTML([]rune(throttleMsg))
	}
	charsLeftForEvents := charsForEvents - charsForThrottleMsg

	var eventsString string
	eventsString += codeBlockStart
	var tailString string

	eventsLenLimitReached := false
	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("\n%s", eventStringFormatter(event, formatter.location))
		if msg := event.CreateMessage(formatter.location); len(msg) > 0 {
			line += fmt.Sprintf(". %s", msg)
		}

		tailString = fmt.Sprintf("\n...and %d more events.", len(events)-eventsPrinted)
		tailStringLen := len("\n") + len([]rune(tailString))
		if !(charsForEvents < 0) && (len([]rune(eventsString))+len([]rune(line)) > charsLeftForEvents-tailStringLen) {
			eventsLenLimitReached = true
			break
		}

		eventsString += line
		eventsPrinted++
	}
	eventsString += "\n"
	eventsString += codeBlockEnd

	if eventsLenLimitReached {
		eventsString += tailString
	}

	if throttled {
		eventsString += throttleMsg
	}

	return eventsString
}

func uriFormatter(triggerURI, triggerName string) string {
	return fmt.Sprintf("<a href=\"%s\">%s</a>", triggerURI, html.EscapeString(triggerName))
}

var (
	startHeaderRegexp = regexp.MustCompile("<h[0-9]+>")
	endHeaderRegexp   = regexp.MustCompile("</h[0-9]+>")
)

func descriptionFormatter(trigger moira.TriggerData) string {
	desc := trigger.Desc
	if trigger.Desc != "" {
		desc += "\n"
	} else {
		return ""
	}

	// Sometimes in trigger description may be text constructions like <param>.
	// blackfriday may recognise it as tag, so it won't be escaped.
	// Then it is sent to telegram we will get error: 'Bad request', because telegram doesn't support such tag.
	replacer := strings.NewReplacer(
		"<", "&lt;",
		">", "&gt;",
	)
	mdWithNoTags := replacer.Replace(desc)

	htmlDescStr := string(blackfriday.Run([]byte(mdWithNoTags),
		blackfriday.WithExtensions(
			blackfriday.CommonExtensions &
				^blackfriday.DefinitionLists &
				^blackfriday.Tables),
		blackfriday.WithRenderer(
			blackfriday.NewHTMLRenderer(
				blackfriday.HTMLRendererParameters{
					Flags: blackfriday.UseXHTML,
				}))))

	// html headers are not supported by telegram html, so make them bold instead.
	htmlDescStr = startHeaderRegexp.ReplaceAllString(htmlDescStr, "<b>")
	replacedHeaders := endHeaderRegexp.ReplaceAllString(htmlDescStr, "</b>")

	// some tags are not supported, so replace them.
	tagReplacer := strings.NewReplacer(
		"<p>", "",
		"</p>", "",
		"<ul>", "",
		"</ul>", "",
		"<li>", "- ",
		"</li>", "",
		"<ol>", "",
		"</ol>", "",
		"<hr>", "",
		"<hr />", "",
		"<br>", "\n")

	return tagReplacer.Replace(replacedHeaders)
}

const (
	tooLongDescMessage = "\n[description is too long for telegram sender]\n"
)

func descriptionCutter(_ string, maxSize int) string {
	if len([]rune(tooLongDescMessage)) < maxSize {
		return tooLongDescMessage
	}

	return ""
}

func boldFormatter(str string) string {
	return fmt.Sprintf("<b>%s</b>", html.EscapeString(str))
}

func eventStringFormatter(event moira.NotificationEvent, loc *time.Location) string {
	return fmt.Sprintf(
		"%s: <code>%s</code> = %s (%s to %s)",
		event.FormatTimestamp(loc, moira.DefaultTimeFormat),
		html.EscapeString(event.Metric),
		html.EscapeString(event.GetMetricsValues(moira.DefaultNotificationSettings)),
		event.OldState,
		event.State)
}
