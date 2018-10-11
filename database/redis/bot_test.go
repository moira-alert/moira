package redis

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"fmt"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/logging/go-logging"
	"time"
)

func TestRenewBotRegistration(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := NewDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	lockTTL := 3000
	lockTime := time.Duration(lockTTL) * time.Millisecond

	var firstLockString string
	var secondLockString string
	var testLockString string
	var err error

	Convey("Manage bot registrations", t, func() {
		Convey("No registrations to renew", func() {
			renewed := dataBase.RenewBotRegistration(messenger3)
			So(renewed, ShouldBeFalse)
		})
		Convey("Just register, should be registered", func() {
			registered := dataBase.RegisterBotIfAlreadyNot(messenger3, lockTime)
			So(registered, ShouldBeTrue)
		})
		Convey("This messenger should be a temp user, with auto generated string", func() {
			firstLockString, err = dataBase.GetIDByUsername(messenger3, botUsername)
			So(err, ShouldBeNil)
			So(firstLockString, ShouldNotBeEmpty)
			fmt.Println(firstLockString)
		})
		Convey("Register second messenger, should be as temp user, with new string", func() {
			lockResults := testLockWithTTLExpireErrorExpected(lockTTL, 3, func() bool {
				return dataBase.RegisterBotIfAlreadyNot(messenger3, lockTime)
			})
			So(lockResults, ShouldContain, true)

			secondLockString, err = dataBase.GetIDByUsername(messenger3, botUsername)
			So(err, ShouldBeNil)
			So(secondLockString, ShouldNotBeEmpty)
			So(firstLockString, ShouldNotResemble, secondLockString)
			fmt.Println(secondLockString)
		})
		Convey("Renew bot registration, should be renewed", func() {
			testLockString, err = dataBase.GetIDByUsername(messenger3, botUsername)
			So(err, ShouldBeNil)
			So(firstLockString, ShouldNotBeEmpty)
			So(secondLockString, ShouldResemble, testLockString)

			renewed := dataBase.RenewBotRegistration(messenger3)
			So(renewed, ShouldBeTrue)
		})
		Convey("Renew bot registration, should not be renewed", func() {
			lockResults := testLockWithTTLExpireErrorExpected(lockTTL, 2, func() bool {
				return dataBase.RenewBotRegistration(messenger3)
			})
			So(lockResults[len(lockResults)-1], ShouldBeFalse)
		})
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

		err = dataBase.RemoveUser(messenger2, user1)
		So(err, ShouldNotBeNil)

		actual2 := dataBase.RegisterBotIfAlreadyNot(messenger3, 30)
		So(actual2, ShouldBeFalse)
	})
}

var messenger1 = "messenger1"
var messenger2 = "messenger2"
var messenger3 = "messenger3"
