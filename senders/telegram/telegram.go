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
	telegramMessageLimit = 4096
	emojiStates          = map[string]string{
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

type recipient struct {
	uid string
}

func (r recipient) Destination() string {
	return r.uid
}

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location) error {
	sender.APIToken = senderSettings["api_token"]
	if sender.APIToken == "" {
		return fmt.Errorf("Can not read telegram api_token from config")
	}
	sender.logger = logger
	sender.FrontURI = senderSettings["front_uri"]
	sender.location = location

	err := sender.StartTelebot()
	if err != nil {
		return fmt.Errorf("Error starting bot: %s", err)
	}
	return nil
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

// StartTelebot creates an api and start telebot
func (sender *Sender) StartTelebot() error {
	ttl := time.Second * 30
	var err error
	sender.bot, err = telebot.NewBot(sender.APIToken)
	if err != nil {
		return err
	}
	firstCheck := true
	go func() {
		for {
			if sender.DataBase.RegisterBotIfAlreadyNot(messenger, ttl) {
				sender.logger.Infof("Registered new %s bot, checking for new messages", messenger)
				go sender.Loop(1 * time.Second)
				sender.renewSubscription(ttl)
				return
			}
			checkingInterval := time.Minute
			if firstCheck {
				sender.logger.Infof("%s bot already registered, trying for register every %v in loop", messenger, checkingInterval)
				firstCheck = false
			}
			<-time.After(checkingInterval)
		}
	}()
	return nil
}

// Loop starts api loop
func (sender *Sender) Loop(timeout time.Duration) {
	messages := make(chan telebot.Message)
	sender.bot.Listen(messages, timeout)

	for {
		message, ok := <-messages
		if !ok {
			sender.logger.Warning("Telegram messages channel was closed, stop listening and deregister")
			sender.DataBase.DeregisterBot(messenger)
			return
		}
		if err := sender.handleMessage(message); err != nil {
			sender.logger.Error("Error sending message")
		}
	}
}

func (sender *Sender) renewSubscription(ttl time.Duration) {
	checkTicker := time.NewTicker((ttl / time.Second) / 2 * time.Second)
	for {
		<-checkTicker.C
		if !sender.DataBase.RenewBotRegistration(messenger) {
			sender.logger.Warningf("Could not renew subscription for %s bot, try to register bot again", messenger)
			if !sender.DataBase.RegisterBotIfAlreadyNot(messenger, ttl) {
				// TODO (borovskyav) here is a bug: if before register bot, another instance will register bot too, we will see two working listeners, and second listener will make errors while receive new messages in loop
				// TODO best way - graceful stop of failed renew listener, and try to register bots again. But telebot has not graceful stopping.
				sender.logger.Errorf("Could not register %s bot again, another instance did it already", messenger)
				return
			}
			sender.logger.Infof("%s bot successfully registered again", messenger)
		}
	}
}

// Talk processes one talk
func (sender *Sender) Talk(username, message string) error {
	uid, err := sender.DataBase.GetIDByUsername(messenger, username)
	if err != nil {
		return fmt.Errorf("failed to get username uuid: %s", err.Error())
	}
	var options *telebot.SendOptions
	return sender.bot.SendMessage(recipient{uid}, message, options)
}

func (sender *Sender) handleMessage(message telebot.Message) error {
	var err error
	var options *telebot.SendOptions
	id := strconv.FormatInt(message.Chat.ID, 10)
	title := message.Chat.Title
	userTitle := strings.Trim(fmt.Sprintf("%s %s", message.Sender.FirstName, message.Sender.LastName), " ")
	username := message.Chat.Username
	chatType := message.Chat.Type
	switch {
	case chatType == "private" && message.Text == "/start":
		if username == "" {
			sender.bot.SendMessage(message.Chat, "Username is empty. Please add username in Telegram.", options)
		} else {
			err = sender.DataBase.SetUsernameID(messenger, "@"+username, id)
			if err != nil {
				return err
			}
			sender.bot.SendMessage(message.Chat, fmt.Sprintf("Okay, %s, your id is %s", userTitle, id), nil)
		}
	case chatType == "supergroup" || chatType == "group":
		uid, _ := sender.DataBase.GetIDByUsername(messenger, title)
		if uid == "" {
			sender.bot.SendMessage(message.Chat, fmt.Sprintf("Hi, all!\nI will send alerts in this group (%s).", title), nil)
		}
		fmt.Println(chatType, title)
		err = sender.DataBase.SetUsernameID(messenger, title, id)
		if err != nil {
			return err
		}
	default:
		sender.bot.SendMessage(message.Chat, "I don't understand you :(", nil)
	}
	return err
}
