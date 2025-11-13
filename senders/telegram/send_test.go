package telegram

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"go.uber.org/mock/gomock"

	"github.com/moira-alert/moira/database"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_telegram "github.com/moira-alert/moira/mock/notifier/telegram"

	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/telebot.v3"
)

func TestGetChat(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	bot := mock_telegram.NewMockBot(mockCtrl)
	sender := Sender{DataBase: dataBase, bot: bot}

	Convey("Get Telegram Chat From DB", t, func() {
		Convey("Compatibility with Moira < 2.12.0", func() {
			Convey("For private chat should fetch from DB", func() {
				idStr := "7824728482"

				dataBase.EXPECT().GetChatByUsername(messenger, "@durov").Return("7824728482", nil)

				id, err := strconv.ParseInt(idStr, 10, 64)
				So(err, ShouldBeNil)

				expected := &Chat{
					ID: id,
				}

				actual, err := sender.getChat("@durov")
				So(actual, ShouldResemble, expected)
				So(err, ShouldBeNil)
			})

			Convey("For supergroup's main thread should fetch from DB", func() {
				idStr := "-1001494975744"
				dataBase.EXPECT().GetChatByUsername(messenger, "somesupergroup / moira").Return(idStr, nil)

				id, err := strconv.ParseInt(idStr, 10, 64)
				So(err, ShouldBeNil)

				expected := &Chat{
					ID: id,
				}

				actual, err := sender.getChat("somesupergroup / moira")
				So(actual, ShouldResemble, expected)
				So(err, ShouldBeNil)
			})

			Convey("If no UID exists in database for this username", func() {
				dataBase.EXPECT().GetChatByUsername(messenger, "@durov").Return("", database.ErrNil)

				actual, err := sender.getChat("@durov")
				So(err, ShouldResemble, fmt.Errorf("failed to get username chat: %w", database.ErrNil))
				So(actual, ShouldBeNil)
			})
		})

		Convey("Moira >= 2.12.0", func() {
			Convey(`For private channel with % prefix should fetch info from Telegram`, func() {
				expectedChat := &telebot.Chat{
					ID:   -1001494975744,
					Type: telebot.ChatPrivate,
				}

				bot.EXPECT().ChatByUsername("-1001494975744").Return(expectedChat, nil)

				actual, err := sender.getChat("%1494975744")
				expected := &Chat{
					ID: -1001494975744,
				}

				So(actual, ShouldResemble, expected)
				So(err, ShouldBeNil)
			})

			Convey("For public channel with # prefix should fetch info from Telegram", func() {
				expectedChat := &telebot.Chat{
					ID:       -1001494975744,
					Type:     telebot.ChatChannel,
					Username: "MyPublicChannel",
				}

				bot.EXPECT().ChatByUsername("@MyPublicChannel").Return(expectedChat, nil)

				actual, err := sender.getChat("#MyPublicChannel")
				expected := &Chat{
					ID: -1001494975744,
				}

				So(actual, ShouldResemble, expected)
				So(err, ShouldBeNil)
			})

			Convey("For private chat should fetch from DB", func() {
				dataBase.EXPECT().GetChatByUsername(messenger, "@durov").Return(`{"chat_id":1}`, nil)

				actual, err := sender.getChat("@durov")
				expected := &Chat{
					ID: 1,
				}

				So(actual, ShouldResemble, expected)
				So(err, ShouldBeNil)
			})

			Convey("For group should fetch from DB", func() {
				dataBase.EXPECT().GetChatByUsername(messenger, "somegroup / moira").Return(`{"chat_id":-1001494975744}`, nil)

				actual, err := sender.getChat("somegroup / moira")
				expected := &Chat{
					ID: -1001494975744,
				}

				So(actual, ShouldResemble, expected)
				So(err, ShouldBeNil)
			})

			Convey("For supergroup's main thread should fetch from DB", func() {
				dataBase.EXPECT().GetChatByUsername(messenger, "somesupergroup / moira").Return(`{"chat_id":-1001494975744}`, nil)

				actual, err := sender.getChat("somesupergroup / moira")
				expected := &Chat{
					ID: -1001494975744,
				}

				So(actual, ShouldResemble, expected)
				So(err, ShouldBeNil)
			})

			Convey("For supergroup's thread should fetch from DB", func() {
				dataBase.EXPECT().GetChatByUsername(messenger, "-1001494975744/10").Return(`{"chat_id":-1001494975744,"thread_id":10}`, nil)

				actual, err := sender.getChat("-1001494975744/10")
				expected := &Chat{
					ID:       -1001494975744,
					ThreadID: 10,
				}

				So(actual, ShouldResemble, expected)
				So(err, ShouldBeNil)
			})

			Convey("If no record exists in database for this contactValue", func() {
				dataBase.EXPECT().GetChatByUsername(messenger, "-1001494975744/20").Return("", database.ErrNil)

				actual, err := sender.getChat("-1001494975744/20")
				So(err, ShouldResemble, fmt.Errorf("failed to get username chat: %w", database.ErrNil))
				So(actual, ShouldBeNil)
			})
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
			for brokenContactError := range brokenContactAPIErrors {
				err := checkBrokenContactError(logger, brokenContactError)
				So(err, ShouldHaveSameTypeAs, moira.SenderBrokenContactError{})

				var convertedErr moira.SenderBrokenContactError

				errors.As(err, &convertedErr)
				So(convertedErr.SenderError, ShouldEqual, brokenContactError)
			}
		})
		Convey("Other errors are returned as is", func() {
			otherTelebotErrors := []*telebot.Error{
				telebot.ErrInternal,
				telebot.ErrTooLarge,
				telebot.ErrEmptyMessage,
				telebot.ErrWrongFileID,
				telebot.ErrNoRightsToDelete,
				telebot.ErrCantRemoveOwner,
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

func TestCheckBadMessageError(t *testing.T) {
	Convey("Check bad message error", t, func() {
		Convey("nil error is nil", func() {
			err, ok := checkBadMessageError(nil)

			So(err, ShouldBeNil)
			So(ok, ShouldBeFalse)
		})

		Convey("proper telebot errors is recognised", func() {
			for givenErr := range badMessageFormatErrors {
				err, ok := checkBadMessageError(givenErr)

				So(err, ShouldEqual, givenErr)
				So(ok, ShouldBeTrue)
			}
		})

		Convey("other telebot errors are not recognised", func() {
			otherErrors := []*telebot.Error{
				telebot.ErrInternal,
				telebot.ErrEmptyMessage,
				telebot.ErrWrongFileID,
				telebot.ErrNoRightsToDelete,
				telebot.ErrCantRemoveOwner,
				telebot.ErrUnauthorized,
				telebot.ErrNoRightsToSendPhoto,
				telebot.ErrChatNotFound,
			}

			for _, otherError := range otherErrors {
				err, ok := checkBadMessageError(otherError)

				So(err, ShouldEqual, otherError)
				So(ok, ShouldBeFalse)
			}
		})

		Convey("errors with proper message is recognised", func() {
			givenErrors := []error{
				fmt.Errorf("telegram: Bad Request: can't parse InputMedia: Can't parse entities: Unsupported start tag \"sup\" at byte offset 396 (400)"),
				fmt.Errorf("telegram: Bad Request: message caption is too long (400)"),
				fmt.Errorf("telegram: Bad Request: can't parse entities: Unsupported start tag \"container_name\" at byte offset 729 (400)"),
			}

			for _, givenErr := range givenErrors {
				err, ok := checkBadMessageError(givenErr)

				So(err, ShouldEqual, givenErr)
				So(ok, ShouldBeTrue)
			}
		})
	})
}
