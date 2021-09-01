package goredis

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira/database"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
)

func TestGetIDByUsername(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test", true)
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Messengers manipulation", t, func() {
		Convey("Get-set usernames", func() {
			Convey("Just set username to one of messengers", func() {
				err := dataBase.SetUsernameID(messenger1, user1, "id1")
				So(err, ShouldBeNil)
			})

			Convey("Check it for existing", func() {
				actual, err := dataBase.GetIDByUsername(messenger1, user1)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, "id1")
			})

			Convey("Check for not existing in two another messengers", func() {
				actual, err := dataBase.GetIDByUsername(messenger2, user1)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldBeEmpty)

				actual, err = dataBase.GetIDByUsername(messenger3, user1)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldBeEmpty)
			})

			Convey("Get username with # prefix should return @username", func() {
				actual, err := dataBase.GetIDByUsername(messenger1, "#"+user1)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, "@"+user1)
			})

			Convey("Remove this user", func() {
				err := dataBase.RemoveUser(messenger1, user1)
				So(err, ShouldBeNil)
			})

			Convey("Check it for unexisting", func() {
				actual, err := dataBase.GetIDByUsername(messenger1, user1)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldBeEmpty)
			})
		})
	})
}

var messenger1 = "messenger1"
var messenger2 = "messenger2"
var messenger3 = "messenger3"
