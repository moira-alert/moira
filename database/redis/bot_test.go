package redis

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/logging/go-logging"
)

func TestBotDataStoring(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Messengers manipulation", t, func(c C) {
		Convey("Get-set usernames", t, func(c C) {
			Convey("Just set username to one of messengers", t, func(c C) {
				err := dataBase.SetUsernameID(messenger1, user1, "id1")
				c.So(err, ShouldBeNil)
			})

			Convey("Check it for existing", t, func(c C) {
				actual, err := dataBase.GetIDByUsername(messenger1, user1)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, "id1")
			})

			Convey("Check for not existing in two another messengers", t, func(c C) {
				actual, err := dataBase.GetIDByUsername(messenger2, user1)
				c.So(err, ShouldResemble, database.ErrNil)
				c.So(actual, ShouldBeEmpty)

				actual, err = dataBase.GetIDByUsername(messenger3, user1)
				c.So(err, ShouldResemble, database.ErrNil)
				c.So(actual, ShouldBeEmpty)
			})

			Convey("Get username with # prefix should return @username", t, func(c C) {
				actual, err := dataBase.GetIDByUsername(messenger1, "#"+user1)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, "@"+user1)
			})

			Convey("Remove this user", t, func(c C) {
				err := dataBase.RemoveUser(messenger1, user1)
				c.So(err, ShouldBeNil)
			})

			Convey("Check it for unexisting", t, func(c C) {
				actual, err := dataBase.GetIDByUsername(messenger1, user1)
				c.So(err, ShouldResemble, database.ErrNil)
				c.So(actual, ShouldBeEmpty)
			})

		})
	})
}

func TestBotDataStoringErrorConnection(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func(c C) {
		actual1, err := dataBase.GetIDByUsername(messenger1, user1)
		c.So(actual1, ShouldBeEmpty)
		c.So(err, ShouldNotBeNil)

		err = dataBase.SetUsernameID(messenger2, user1, "id1")
		c.So(err, ShouldNotBeNil)

		err = dataBase.RemoveUser(messenger2, user1)
		c.So(err, ShouldNotBeNil)
	})
}

var messenger1 = "messenger1"
var messenger2 = "messenger2"
var messenger3 = "messenger3"
