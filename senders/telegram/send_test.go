package telegram

import (
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/tucnak/telebot.v2"
)

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
				var convertedErr moira.SenderBrokenContactError
				errors.As(err, &convertedErr)
				So(convertedErr.SenderError, ShouldEqual, brokenContactError)
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
			var convertedErr moira.SenderBrokenContactError
			errors.As(err, &convertedErr)
			So(convertedErr.SenderError, ShouldEqual, userNotFoundError)
		})
	})
}
