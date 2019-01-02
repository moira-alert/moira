package redis

import (
	"github.com/moira-alert/moira"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/logging/go-logging"
)

//func TestRenewBot(t *testing.T) {
//	logger, _ := logging.ConfigureLog("stdout", "info", "test")
//	dataBase := newTestDatabase(logger, config)
//	dataBase.flush()
//	//defer dataBase.flush()
//
//	Convey("Renew bot", t, func() {
//		extended := dataBase.RenewBotRegistration(messenger3)
//		So(extended, ShouldBeFalse)
//
//		actual := dataBase.RegisterBotIfAlreadyNot(messenger3, time.Second*3)
//		So(actual, ShouldBeTrue)
//
//		firstLockString, _ := dataBase.GetIDByUsername(messenger3, botUsername)
//		fmt.Println(firstLockString)
//		So(firstLockString, ShouldNotBeEmpty)
//
//		time.Sleep(time.Second * 3)
//
//		// extended = dataBase.RenewBotRegistration(messenger3)
//		// So(extended, ShouldBeFalse)
//
//		actual = dataBase.RegisterBotIfAlreadyNot(messenger3, time.Second*3)
//		So(actual, ShouldBeTrue)
//
//		secondLockString, _ := dataBase.GetIDByUsername(messenger3, botUsername)
//		fmt.Println(secondLockString)
//		So(firstLockString, ShouldNotBeEmpty)
//		So(firstLockString, ShouldNotResemble, secondLockString)
//
//		time.Sleep(time.Millisecond * 1500)
//
//		secondLockString1, _ := dataBase.GetIDByUsername(messenger3, botUsername)
//		So(firstLockString, ShouldNotBeEmpty)
//		So(secondLockString, ShouldResemble, secondLockString1)
//
//		extended = dataBase.RenewBotRegistration(messenger3)
//		So(extended, ShouldBeTrue)
//
//		time.Sleep(time.Second * 2)
//
//		secondLockString2, _ := dataBase.GetIDByUsername(messenger3, botUsername)
//		So(firstLockString, ShouldNotBeEmpty)
//		So(secondLockString1, ShouldResemble, secondLockString2)
//
//		time.Sleep(time.Second * 2)
//
//		extended = dataBase.RenewBotRegistration(messenger3)
//		So(extended, ShouldBeFalse)
//	})
//}

func TestBotDataStoring(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
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

func TestBotDataStoringErrorConnection(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test")
	dataBase := newTestDatabase(logger, emptyConfig)
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

		actual2 := dataBase.RegisterServiceIfNotDone(messenger3Registration, 30)
		So(actual2, ShouldBeFalse)
	})
}

var messenger1 = "messenger1"
var messenger2 = "messenger2"
var messenger3 = "messenger3"
var messenger3Registration moira.SingleInstanceService = "notifier:messenger3"
