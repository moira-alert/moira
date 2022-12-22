package telegram

import (
	"fmt"
	"testing"
	"time"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/tucnak/telebot.v2"
)

func TestBuildMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{location: location, frontURI: "http://moira.url"}

	Convey("Build Moira Message tests", t, func() {
		event := moira.NotificationEvent{
			TriggerID: "TriggerID",
			Values:    map[string]float64{"t1": 97.4458331200185},
			Timestamp: 150000000,
			Metric:    "Metric name",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
		}

		trigger := moira.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Trigger Name",
			ID:   "TriggerID",
		}

		Convey("Print moira message with one event", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false, messageMaxCharacters)
			expected := `💣NODATA Trigger Name [tag1][tag2] (1)

02:40: Metric name = 97.4458331200185 (OK to NODATA)

http://moira.url/trigger/TriggerID
`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty triggerID, but with trigger Name", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{Name: "Name"}, false, messageMaxCharacters)
			expected := `💣NODATA Name  (1)

02:40: Metric name = 97.4458331200185 (OK to NODATA)`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty trigger", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{}, false, messageMaxCharacters)
			expected := `💣NODATA   (1)

02:40: Metric name = 97.4458331200185 (OK to NODATA)`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			event.TriggerID = ""
			trigger.ID = ""
			var interval int64 = 24
			event.MessageEventInfo = &moira.EventInfo{Interval: &interval}
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, false, messageMaxCharacters)
			expected := `💣NODATA Trigger Name [tag1][tag2] (1)

02:40: Metric name = 97.4458331200185 (OK to NODATA). This metric has been in bad state for more than 24 hours - please, fix.`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, trigger, true, messageMaxCharacters)
			expected := `💣NODATA Trigger Name [tag1][tag2] (1)

02:40: Metric name = 97.4458331200185 (OK to NODATA)

http://moira.url/trigger/TriggerID

Please, fix your system or tune this trigger to generate less events.`
			So(actual, ShouldResemble, expected)
		})

		events := make([]moira.NotificationEvent, 0)
		Convey("Print moira message with 6 events and photo message length", func() {
			for i := 0; i < 18; i++ {
				events = append(events, event)
			}
			actual := sender.buildMessage(events, trigger, false, albumCaptionMaxCharacters)
			expected := `💣NODATA Trigger Name [tag1][tag2] (18)

02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)

...and 7 more events.

http://moira.url/trigger/TriggerID
`
			fmt.Printf("Bytes: %v\n", len(expected))
			fmt.Printf("Symbols: %v\n", len([]rune(expected)))
			So(actual, ShouldResemble, expected)
		})
	})
}

func TestGetChatUID(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	sender := Sender{location: location, frontURI: "http://moira.url", DataBase: dataBase}

	Convey("Get Telegram chat's UID", t, func() {
		Convey("For private channel with % prefix should return with -100 prefix", func() {
			actual, err := sender.getChatUID("%1494975744")
			expected := "-1001494975744"
			So(actual, ShouldResemble, expected)
			So(err, ShouldBeNil)
		})

		Convey("For public channel with # prefix should return with @ prefix", func() {
			dataBase.EXPECT().GetIDByUsername(messenger, "#MyPublicChannel").Return("@MyPublicChannel", nil)
			actual, err := sender.getChatUID("#MyPublicChannel")
			expected := "@MyPublicChannel"
			So(actual, ShouldResemble, expected)
			So(err, ShouldBeNil)
		})

		Convey("If no UID exists in database for this username", func() {
			dataBase.EXPECT().GetIDByUsername(messenger, "@durov").Return("", database.ErrNil)
			actual, err := sender.getChatUID("@durov")
			So(err, ShouldResemble, fmt.Errorf("failed to get username uuid: nil returned"))
			So(actual, ShouldBeEmpty)
		})
	})
}

func TestPrepareAlbum(t *testing.T) {
	Convey("Prepare album", t, func() {
		Convey("Only the first photo of the album has a caption", func() {
			Convey("An album with one photo", func() {
				plots := [][]byte{{1, 0, 1}}
				album := prepareAlbum(plots, "caption")

				So(album[0].(*telebot.Photo).Caption, ShouldEqual, "caption")
			})

			Convey("An album with several photos", func() {
				plots := [][]byte{{1, 0, 1}, {1, 0, 0}, {0, 0, 1}}
				album := prepareAlbum(plots, "caption")

				So(album[0].(*telebot.Photo).Caption, ShouldEqual, "caption")
				So(album[1].(*telebot.Photo).Caption, ShouldEqual, "")
				So(album[2].(*telebot.Photo).Caption, ShouldEqual, "")
			})
		})
	})
}

func TestCheckBrokenContactError(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "warn", "test", true)
	Convey("Check broken contact error", t, func() {
		Convey("Nil error is nil", func() {
			err := checkBrokenContactError(logger, nil)
			So(err, ShouldBeNil)
		})
		Convey("Broken contact error is properly recognized", func() {
			brokenContactErrorsList := []*telebot.APIError{
				telebot.ErrNoRightsToSendPhoto,
				telebot.ErrChatNotFound,
				telebot.ErrNoRightsToSend,
				telebot.ErrUnauthorized,
				telebot.ErrBlockedByUser,
				telebot.ErrUserIsDeactivated,
				telebot.ErrBotKickedFromGroup,
				telebot.ErrBotKickedFromSuperGroup,
			}
			for _, brokenContactError := range brokenContactErrorsList {
				err := checkBrokenContactError(logger, brokenContactError)
				So(err, ShouldHaveSameTypeAs, moira.SenderBrokenContactError{})
				So(err.(moira.SenderBrokenContactError).SenderError, ShouldEqual, brokenContactError)
			}
		})
		Convey("Other errors are returned as is", func() {
			otherTelebotErrors := []*telebot.APIError{
				telebot.ErrInternal,
				telebot.ErrTooLarge,
				telebot.ErrEmptyMessage,
				telebot.ErrWrongFileID,
				telebot.ErrNoRightsToDelete,
				telebot.ErrKickingChatOwner,
			}
			for _, otherError := range otherTelebotErrors {
				err := checkBrokenContactError(logger, otherError)
				So(err, ShouldEqual, otherError)
			}
		})
		Convey("Error on getting username is broken contact error", func() {
			userNotFoundError := fmt.Errorf("failed to get username uuid: nil returned")
			err := checkBrokenContactError(logger, userNotFoundError)
			So(err, ShouldHaveSameTypeAs, moira.SenderBrokenContactError{})
			So(err.(moira.SenderBrokenContactError).SenderError, ShouldEqual, userNotFoundError)
		})
	})
}
