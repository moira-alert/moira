package redis

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"fmt"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/logging/go-logging"
	"time"
)

func TestR(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := NewDatabase(logger, config)
	dataBase.flush()
	//defer dataBase.flush()

	Convey("Renew bot", t, func() {
		extended := dataBase.RenewBotRegistration(messenger3)
		So(extended, ShouldBeFalse)

		actual := dataBase.RegisterBotIfAlreadyNot(messenger3, time.Second*3)
		So(actual, ShouldBeTrue)

		firstLockString, _ := dataBase.GetIDByUsername(messenger3, botUsername)
		fmt.Println(firstLockString)
		So(firstLockString, ShouldNotBeEmpty)

		time.Sleep(time.Second * 3)

		extended = dataBase.RenewBotRegistration(messenger3)
		So(extended, ShouldBeFalse)

		actual = dataBase.RegisterBotIfAlreadyNot(messenger3, time.Second*3)
		So(actual, ShouldBeTrue)

		secondLockString, _ := dataBase.GetIDByUsername(messenger3, botUsername)
		fmt.Println(secondLockString)
		So(firstLockString, ShouldNotBeEmpty)
		So(firstLockString, ShouldNotResemble, secondLockString)

		time.Sleep(time.Millisecond * 1500)

		secondLockString1, _ := dataBase.GetIDByUsername(messenger3, botUsername)
		So(firstLockString, ShouldNotBeEmpty)
		So(secondLockString, ShouldResemble, secondLockString1)

		extended = dataBase.RenewBotRegistration(messenger3)
		So(extended, ShouldBeTrue)

		time.Sleep(time.Second * 2)

		secondLockString2, _ := dataBase.GetIDByUsername(messenger3, botUsername)
		So(firstLockString, ShouldNotBeEmpty)
		So(secondLockString1, ShouldResemble, secondLockString2)

		time.Sleep(time.Second * 2)

		extended = dataBase.RenewBotRegistration(messenger3)
		So(extended, ShouldBeFalse)
	})
}

func TestBotDataStoring(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := NewDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Messengers manipulation", t, func() {
		Convey("Register-deregister messenger", func() {
			Convey("Nothing to deregister", func() {
				unlocked := dataBase.DeregisterBot(messenger2)
				So(unlocked, ShouldBeFalse)
			})

			Convey("Just register, should be registered", func() {
				actual := dataBase.RegisterBotIfAlreadyNot(messenger2, time.Second*30)
				So(actual, ShouldBeTrue)
			})

			var firstLockString string
			Convey("This messenger should be a temp user, with auto generated string", func() {
				firstLockString, _ = dataBase.GetIDByUsername(messenger2, botUsername)
				fmt.Println(firstLockString)
				So(firstLockString, ShouldNotBeEmpty)
			})

			Convey("Register same messenger, should be not registered", func() {
				actual := dataBase.RegisterBotIfAlreadyNot(messenger2, time.Second*30)
				So(actual, ShouldBeFalse)
			})

			Convey("DeregisterBot should deregister it", func() {
				unlocked := dataBase.DeregisterBot(messenger2)
				So(unlocked, ShouldBeTrue)
			})

			Convey("And Register it again, should be as temp user, with new string", func() {
				actual := dataBase.RegisterBotIfAlreadyNot(messenger2, time.Second*30)
				So(actual, ShouldBeTrue)

				secondLockString, err := dataBase.GetIDByUsername(messenger2, botUsername)
				fmt.Println(secondLockString)
				So(err, ShouldBeNil)
				So(secondLockString, ShouldNotBeEmpty)
				So(firstLockString, ShouldNotResemble, secondLockString)
			})

			Convey("Now deregister it via DeregisterBots and check for nil returned", func() {
				dataBase.DeregisterBots()
				actual, err := dataBase.GetIDByUsername(messenger2, botUsername)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldBeEmpty)
			})
		})

		Convey("Register-deregister several messengers", func() {
			dataBase.flush()

			actual := dataBase.RegisterBotIfAlreadyNot(messenger1, time.Second*30)
			So(actual, ShouldBeTrue)
			actual = dataBase.RegisterBotIfAlreadyNot(messenger2, time.Second*30)
			So(actual, ShouldBeTrue)
			actual = dataBase.RegisterBotIfAlreadyNot(messenger3, time.Second*30)
			So(actual, ShouldBeTrue)

			Convey("All messengers should have temp user, with host name", func() {
				actual, err := dataBase.GetIDByUsername(messenger1, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldNotBeEmpty)

				actual, err = dataBase.GetIDByUsername(messenger2, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldNotBeEmpty)

				actual, err = dataBase.GetIDByUsername(messenger3, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldNotBeEmpty)
			})

			Convey("Now deregister one of messenges via DeregisterBot and check for deregistered flag and hostname in another", func() {
				dataBase.DeregisterBot(messenger3)
				actual, err := dataBase.GetIDByUsername(messenger3, botUsername)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldBeEmpty)

				actual, err = dataBase.GetIDByUsername(messenger1, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldNotBeEmpty)

				actual, err = dataBase.GetIDByUsername(messenger2, botUsername)
				So(err, ShouldBeNil)
				So(actual, ShouldNotBeEmpty)
			})

			Convey("Now call DeregisterBots and check two another for deregistered flag", func() {
				dataBase.DeregisterBots()
				actual, err := dataBase.GetIDByUsername(messenger1, botUsername)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldBeEmpty)
				actual, err = dataBase.GetIDByUsername(messenger2, botUsername)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldBeEmpty)
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

		actual2 := dataBase.RegisterBotIfAlreadyNot(messenger3, 30)
		So(actual2, ShouldBeFalse)
	})
}

var messenger1 = "messenger1"
var messenger2 = "messenger2"
var messenger3 = "messenger3"
