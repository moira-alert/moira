package telegram

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tucnak/telebot"

	"github.com/moira-alert/moira"
)

const messenger = "telegram"

var (
	telegramMessageLimit    = 4096
	pollerTimeout           = 10 * time.Second
	databaseMutexExpiry     = 30 * time.Second
	singlePollerStateExpiry = time.Minute
	emojiStates             = map[string]string{
		"OK":     "\xe2\x9c\x85",
		"WARN":   "\xe2\x9a\xa0",
		"ERROR":  "\xe2\xad\x95",
		"NODATA": "\xf0\x9f\x92\xa3",
		"TEST":   "\xf0\x9f\x98\x8a",
	}
)

// Sender implements moira sender interface via telegram
type Sender struct {
	DataBase moira.Database
	APIToken string
	FrontURI string
	logger   moira.Logger
	bot      *telebot.Bot
	location *time.Location
}

// Init loads yaml config, configures and starts telegram bot
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location) error {
	var err error
	sender.APIToken = senderSettings["api_token"]
	if sender.APIToken == "" {
		return fmt.Errorf("Can not read telegram api_token from config")
	}
	sender.FrontURI = senderSettings["front_uri"]
	sender.logger = logger
	sender.location = location

	sender.bot, err = telebot.NewBot(telebot.Settings{
		Token:  sender.APIToken,
		Poller: &telebot.LongPoller{Timeout: pollerTimeout},
	})
	if err != nil {
		return err
	}

	sender.bot.Handle(telebot.OnText, func(message *telebot.Message) {
		if err = sender.handleMessage(message); err != nil {
			sender.logger.Errorf("Error handling incoming message: %s", err.Error())
		}
	})

	err = sender.RunTelebot()
	if err != nil {
		return fmt.Errorf("Error running bot: %s", err.Error())
	}
	return nil
}

// RunTelebot starts telegam bot and manages bot subscriptions
// to make sure there is always only one working Poller
func (sender *Sender) RunTelebot() error {
	firstCheck := true
	go func() {
		for {
			if sender.DataBase.RegisterBotIfAlreadyNot(messenger, databaseMutexExpiry) {
				sender.logger.Infof("Registered new %s bot, checking for new messages", messenger)
				go sender.bot.Start()
				sender.renewSubscription(databaseMutexExpiry)
				continue
			}
			if firstCheck {
				sender.logger.Infof("%s bot already registered, trying for register every %v in loop", messenger, singlePollerStateExpiry)
				firstCheck = false
			}
			<-time.After(singlePollerStateExpiry)
		}
	}()
	return nil
}

// renewSubscription tries to renew bot subscription
// and gracefully stops bot on fail to prevent multiple Poller instances running
func (sender *Sender) renewSubscription(ttl time.Duration) {
	checkTicker := time.NewTicker((ttl / time.Second) / 2 * time.Second)
	for {
		<-checkTicker.C
		if !sender.DataBase.RenewBotRegistration(messenger) {
			sender.logger.Warningf("Could not renew subscription for %s bot, try to register bot again", messenger)
			sender.bot.Stop()
			return
		}
	}
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, throttled bool) error {

	var message bytes.Buffer

	state := events.GetSubjectState()
	tags := trigger.GetTags()

	emoji := emojiStates[state]
	message.WriteString(fmt.Sprintf("%s%s %s %s (%d)\n", emoji, state, trigger.Name, tags, len(events)))

	messageLimitReached := false
	lineCount := 0

	for _, event := range events {
		value := strconv.FormatFloat(moira.UseFloat64(event.Value), 'f', -1, 64)
		eventTime := time.Unix(event.Timestamp, 0).In(sender.location)
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", eventTime.Format("15:04"), event.Metric, value, event.OldState, event.State)
		if len(moira.UseString(event.Message)) > 0 {
			line += fmt.Sprintf(". %s", moira.UseString(event.Message))
		}
		if message.Len()+len(line) > telegramMessageLimit-400 {
			messageLimitReached = true
			break
		}
		message.WriteString(line)
		lineCount++
	}

	if messageLimitReached {
		message.WriteString(fmt.Sprintf("\n\n...and %d more events.", len(events)-lineCount))
	}

	message.WriteString(fmt.Sprintf("\n\n%s/trigger/%s\n", sender.FrontURI, events[0].TriggerID))

	if throttled {
		message.WriteString("\nPlease, fix your system or tune this trigger to generate less events.")
	}

	sender.logger.Debugf("Calling telegram api with chat_id %s and message body %s", contact.Value, message.String())

	if err := sender.Talk(contact.Value, message.String()); err != nil {
		return fmt.Errorf("Failed to send message to telegram contact %s: %s. ", contact.Value, err)
	}
	return nil

}

// handleMessage handles incoming messages to start sending events to subscribers chats
func (sender *Sender) handleMessage(message *telebot.Message) error {
	var err error
	id := strconv.FormatInt(message.Chat.ID, 10)
	title := message.Chat.Title
	userTitle := strings.Trim(fmt.Sprintf("%s %s", message.Sender.FirstName, message.Sender.LastName), " ")
	username := message.Chat.Username
	chatType := message.Chat.Type
	switch {
	case chatType == "private" && message.Text == "/start":
		if username == "" {
			sender.bot.Send(message.Chat, "Username is empty. Please add username in Telegram.")
		} else {
			err = sender.DataBase.SetUsernameID(messenger, "@"+username, id)
			if err != nil {
				return err
			}
			sender.bot.Send(message.Chat, fmt.Sprintf("Okay, %s, your id is %s", userTitle, id))
		}
	case chatType == "supergroup" || chatType == "group":
		uid, _ := sender.DataBase.GetIDByUsername(messenger, title)
		if uid == "" {
			sender.bot.Send(message.Chat, fmt.Sprintf("Hi, all!\nI will send alerts in this group (%s).", title))
		}
		fmt.Println(chatType, title)
		err = sender.DataBase.SetUsernameID(messenger, title, id)
		if err != nil {
			return err
		}
	default:
		sender.bot.Send(message.Chat, "I don't understand you :(")
	}
	return err
}

// Talk processes one talk
func (sender *Sender) Talk(username, message string) error {
	var err error
	uid, err := sender.DataBase.GetIDByUsername(messenger, username)
	if err != nil {
		return fmt.Errorf("failed to get username uuid: %s", err.Error())
	}
	chat, err := sender.bot.ChatByID(uid)
	if err != nil {
		return fmt.Errorf("can't find recepient %s: %s", uid, err.Error())
	}
	_, err = sender.bot.Send(chat, message)
	if err != nil {
		return fmt.Errorf("can't send message [%s] to %s: %s", message, uid, err.Error())
	}
	return nil
}
