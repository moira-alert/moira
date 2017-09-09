package redis

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira-alert/logging/go-logging"
	"github.com/moira-alert/moira-alert/database"
)

func TestBotDataStoring(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := NewDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Messengers manipulation", t, func() {
		Convey("Register-deregister messenger", func() {
			Convey("Just register, should be registered", func() {
				actual := dataBase.RegisterBotIfAlreadyNot(messenger2)
				So(actual, ShouldBeTrue)
			})

			Convey("This messenger should be as temp user, with host name", func() {
				actual, err := dataBase.GetIDByUsername(messenger2, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, host)
			})

			Convey("Register same messenger, should be registered again", func() {
				actual := dataBase.RegisterBotIfAlreadyNot(messenger2)
				So(actual, ShouldBeTrue)
			})

			Convey("DeregisterBot should deregister it", func() {
				err := dataBase.DeregisterBot(messenger2)
				So(err, ShouldBeNil)
			})

			Convey("Now this messenger temp user should contain deregistered flag", func() {
				actual, err := dataBase.GetIDByUsername(messenger2, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, deregistered)
			})

			Convey("Try to deregister angain, should nothing happen", func() {
				err := dataBase.DeregisterBot(messenger2)
				So(err, ShouldBeNil)
			})

			Convey("And Register it again, should be as temp user, with host name", func() {
				actual := dataBase.RegisterBotIfAlreadyNot(messenger2)
				So(actual, ShouldBeTrue)

				actual1, err := dataBase.GetIDByUsername(messenger2, botUsername)
				So(err, ShouldBeNil)
				So(actual1, ShouldResemble, host)
			})

			Convey("Now deregister it via DeregisterBots and check for deregistered flag", func() {
				dataBase.DeregisterBots()
				actual, err := dataBase.GetIDByUsername(messenger2, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, deregistered)
			})
		})

		Convey("Register-deregister several messengers", func() {
			dataBase.flush()

			actual := dataBase.RegisterBotIfAlreadyNot(messenger1)
			So(actual, ShouldBeTrue)
			actual = dataBase.RegisterBotIfAlreadyNot(messenger2)
			So(actual, ShouldBeTrue)
			actual = dataBase.RegisterBotIfAlreadyNot(messenger3)
			So(actual, ShouldBeTrue)

			Convey("All messengers should have temp user, with host name", func() {
				actual, err := dataBase.GetIDByUsername(messenger1, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, host)

				actual, err = dataBase.GetIDByUsername(messenger2, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, host)

				actual, err = dataBase.GetIDByUsername(messenger3, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, host)
			})

			Convey("Now deregister one of messenges via DeregisterBot and check for deregistered flag and hostname in another", func() {
				dataBase.DeregisterBot(messenger3)
				actual, err := dataBase.GetIDByUsername(messenger3, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, deregistered)

				actual, err = dataBase.GetIDByUsername(messenger1, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, host)

				actual, err = dataBase.GetIDByUsername(messenger2, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, host)
			})

			Convey("Now call DeregisterBots and check two another for deregistered flag", func() {
				dataBase.DeregisterBots()
				actual, err := dataBase.GetIDByUsername(messenger1, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, deregistered)
				actual, err = dataBase.GetIDByUsername(messenger2, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, deregistered)
			})
		})

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

			Convey("Check for not existing in two another messengers", func(){
				actual, err := dataBase.GetIDByUsername(messenger2, user1)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldBeEmpty)

				actual, err = dataBase.GetIDByUsername(messenger3, user1)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldBeEmpty)
			})

			Convey("Get username with # prefix should return @username", func(){
				actual, err := dataBase.GetIDByUsername(messenger1, "#" + user1)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, "@"+user1)
			})
		})
	})
}

func TestBotDataStoringErrorConnection(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := NewDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func() {
		actual1, err := dataBase.GetIDByUsername(messenger1, user1)
		So(actual1, ShouldBeEmpty)
		So(err, ShouldNotBeNil)

		err = dataBase.SetUsernameID(messenger2, user1, "id1")
		So(err, ShouldNotBeNil)

		actual2 := dataBase.RegisterBotIfAlreadyNot(messenger3)
		So(actual2, ShouldBeFalse)
	})
}

var host, _ = os.Hostname()

var messenger1 = "messenger1"
var messenger2 = "messenger2"
var messenger3 = "messenger3"
