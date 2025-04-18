package redis

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira/database"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
)

func TestBotDataStoring(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test", true)
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()

	defer dataBase.Flush()

	Convey("Messengers manipulation", t, func() {
		Convey("Get-set usernames", func() {
			Convey("Just set username to one of messengers", func() {
				err := dataBase.SetUsernameChat(messenger1, user1, "id1")
				So(err, ShouldBeNil)
			})

			Convey("Check it for existing", func() {
				actual, err := dataBase.GetChatByUsername(messenger1, user1)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, "id1")
			})

			Convey("Check for not existing in two another messengers", func() {
				actual, err := dataBase.GetChatByUsername(messenger2, user1)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldBeEmpty)

				actual, err = dataBase.GetChatByUsername(messenger3, user1)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldBeEmpty)
			})

			Convey("Get username with # prefix should return @username", func() {
				actual, err := dataBase.GetChatByUsername(messenger1, "#"+user1)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, "@"+user1)
			})

			Convey("Remove this user", func() {
				err := dataBase.RemoveUser(messenger1, user1)
				So(err, ShouldBeNil)
			})

			Convey("Check it for unexisting", func() {
				actual, err := dataBase.GetChatByUsername(messenger1, user1)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldBeEmpty)
			})
		})
	})
}

func TestBotDataStoringErrorConnection(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test", true)
	dataBase := NewTestDatabaseWithIncorrectConfig(logger)
	dataBase.Flush()

	defer dataBase.Flush()
	Convey("Should throw error when no connection", t, func() {
		actual1, err := dataBase.GetChatByUsername(messenger1, user1)
		So(actual1, ShouldBeEmpty)
		So(err, ShouldNotBeNil)

		err = dataBase.SetUsernameChat(messenger2, user1, "id1")
		So(err, ShouldNotBeNil)

		err = dataBase.RemoveUser(messenger2, user1)
		So(err, ShouldNotBeNil)
	})
}

var (
	messenger1 = "messenger1"
	messenger2 = "messenger2"
	messenger3 = "messenger3"
)
