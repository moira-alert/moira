package redis

import (
	"fmt"
	"testing"

	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

var user1 = "user1"
var user2 = "user2"

func TestContacts(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Contacts manipulation", t, func() {
		Convey("While no data then get contacts should be empty", func() {
			Convey("GetAllContacts should be empty", func() {
				actual1, err := dataBase.GetAllContacts()
				So(err, ShouldBeNil)
				So(actual1, ShouldHaveLength, 0)
			})

			Convey("GetUserContactIDs should be empty", func() {
				actual1, err := dataBase.GetUserContactIDs(user1)
				So(err, ShouldBeNil)
				So(actual1, ShouldHaveLength, 0)

				actual2, err := dataBase.GetUserContactIDs(user2)
				So(err, ShouldBeNil)
				So(actual2, ShouldHaveLength, 0)
			})

			Convey("GetContacts should be empty", func() {
				actual1, err := dataBase.GetContacts([]string{user1Contacts[0].ID, user2Contacts[1].ID})
				So(err, ShouldBeNil)
				So(actual1, ShouldHaveLength, 2)
				for _, contact := range actual1 {
					So(contact, ShouldBeNil)
				}
			})

			Convey("GetContact should be empty", func() {
				actual1, err := dataBase.GetContact(user1Contacts[0].ID)
				So(err, ShouldResemble, database.ErrNil)
				So(actual1, ShouldResemble, moira.ContactData{})
			})
		})

		Convey("Write all contacts for user1 and check it for success write", func() {
			ids := make([]string, len(user1Contacts))
			for i, contact := range user1Contacts {
				ids[i] = contact.ID
				Convey(fmt.Sprintf("Write contact %s and try read", contact.ID), func() {
					err := dataBase.SaveContact(contact)
					So(err, ShouldBeNil)

					actual, err := dataBase.GetContact(contact.ID)
					So(err, ShouldBeNil)
					So(actual, ShouldResemble, *contact)
				})
			}

			Convey("Read all contacts by id", func() {
				actual, err := dataBase.GetContacts(ids)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, user1Contacts)
			})

			Convey("Read all user contacts ids", func() {
				actual, err := dataBase.GetUserContactIDs(user1)
				So(err, ShouldBeNil)
				So(actual, ShouldHaveLength, len(ids))
			})

			Convey("Get all contacts", func() {
				actual, err := dataBase.GetAllContacts()
				So(err, ShouldBeNil)
				So(actual, ShouldHaveLength, len(ids))
			})
		})

		Convey("Write and remove user2 contacts by different strategies", func() {
			ids := make([]string, len(user2Contacts))
			for i, contact := range user2Contacts {
				ids[i] = contact.ID
			}

			contact1 := user2Contacts[0]
			Convey("Save-write contact", func() {
				Convey("Save contact", func() {
					err := dataBase.SaveContact(contact1)
					So(err, ShouldBeNil)

					actual, err := dataBase.GetContact(contact1.ID)
					So(err, ShouldBeNil)
					So(actual, ShouldResemble, *contact1)

					actual1, err := dataBase.GetUserContactIDs(user2)
					So(err, ShouldBeNil)
					So(actual1, ShouldResemble, []string{contact1.ID})
				})

				Convey("Check contacts by read set of contacts", func() {
					actual, err := dataBase.GetContacts(ids)
					So(err, ShouldBeNil)
					So(actual, ShouldHaveLength, len(ids))
					expected := make([]*moira.ContactData, len(ids))
					expected[0] = contact1
					So(actual, ShouldResemble, expected)
				})
			})

			contact2 := user2Contacts[1]
			Convey("Save-remove contact", func() {
				Convey("Just save new contact", func() {
					err := dataBase.SaveContact(contact2)
					So(err, ShouldBeNil)
				})

				Convey("Check it for existence", func() {
					actual, err := dataBase.GetContact(contact2.ID)
					So(err, ShouldBeNil)
					So(actual, ShouldResemble, *contact2)

					actual1, err := dataBase.GetUserContactIDs(user2)
					So(err, ShouldBeNil)
					So(actual1, ShouldHaveLength, 2)
				})

				Convey("Remove contact", func() {
					err := dataBase.RemoveContact(contact2.ID)
					So(err, ShouldBeNil)
				})

				Convey("Check it for not existence", func() {
					actual, err := dataBase.GetContact(contact2.ID)
					So(err, ShouldResemble, database.ErrNil)
					So(actual, ShouldResemble, moira.ContactData{})

					actual1, err := dataBase.GetUserContactIDs(user2)
					So(err, ShouldBeNil)
					So(actual1, ShouldHaveLength, 1)
					So(actual1, ShouldResemble, []string{contact1.ID})

					actual2, err := dataBase.GetContacts(ids)
					expected := make([]*moira.ContactData, len(ids))
					expected[0] = contact1
					So(err, ShouldBeNil)
					So(actual2, ShouldResemble, expected)
				})

				Convey("And save again...", func() {
					err := dataBase.SaveContact(contact2)
					So(err, ShouldBeNil)
				})

				Convey("And check it for existence again", func() {
					actual, err := dataBase.GetContact(contact2.ID)
					So(err, ShouldBeNil)
					So(actual, ShouldResemble, *contact2)

					actual1, err := dataBase.GetUserContactIDs(user2)
					So(err, ShouldBeNil)
					So(actual1, ShouldHaveLength, 2)

					actual2, err := dataBase.GetContacts(ids)
					expected := make([]*moira.ContactData, len(ids))
					expected[0] = contact1
					expected[1] = contact2
					So(err, ShouldBeNil)
					So(actual2, ShouldResemble, expected)
				})
			})

			contact3 := *user2Contacts[2]
			contact3.User = user1
			Convey("Update contact with another user", func() {
				Convey("Just save new contact with user1", func() {
					err := dataBase.SaveContact(&contact3)
					So(err, ShouldBeNil)
				})

				Convey("Check it for existence in user1 contacts", func() {
					actual, err := dataBase.GetContact(contact3.ID)
					So(err, ShouldBeNil)
					So(actual, ShouldResemble, contact3)

					actual1, err := dataBase.GetUserContactIDs(user2)
					So(err, ShouldBeNil)
					So(actual1, ShouldHaveLength, 2)

					actual2, err := dataBase.GetUserContactIDs(user1)
					So(err, ShouldBeNil)
					So(actual2, ShouldHaveLength, 5)
				})

				contact3.User = user2

				Convey("Now save it with user2", func() {
					err := dataBase.SaveContact(&contact3)
					So(err, ShouldBeNil)
				})

				Convey("Check it for existence in user2 contacts and now existence in user1 contacts", func() {
					actual, err := dataBase.GetContact(contact3.ID)
					So(err, ShouldBeNil)
					So(actual, ShouldResemble, contact3)

					actual1, err := dataBase.GetUserContactIDs(user2)
					So(err, ShouldBeNil)
					So(actual1, ShouldHaveLength, 3)

					actual2, err := dataBase.GetUserContactIDs(user1)
					So(err, ShouldBeNil)
					So(actual2, ShouldHaveLength, 4)
				})
			})

			contact4 := user2Contacts[3]
			Convey("Save-update contact", func() {
				Convey("Just save new contact", func() {
					err := dataBase.SaveContact(contact4)
					So(err, ShouldBeNil)
				})

				Convey("Check it for existence", func() {
					actual, err := dataBase.GetContact(contact4.ID)
					So(err, ShouldBeNil)
					So(actual, ShouldResemble, *contact4)

					actual1, err := dataBase.GetUserContactIDs(user2)
					So(err, ShouldBeNil)
					So(actual1, ShouldHaveLength, 4)
				})

				contact2Changed := *contact4
				contact2Changed.Value = "new@email.com"

				Convey("Save updated contact data", func() {
					err := dataBase.SaveContact(&contact2Changed)
					So(err, ShouldBeNil)
				})

				Convey("Check it for new data", func() {
					actual, err := dataBase.GetContact(contact2Changed.ID)
					So(err, ShouldBeNil)
					So(actual, ShouldResemble, contact2Changed)

					actual1, err := dataBase.GetUserContactIDs(user2)
					So(err, ShouldBeNil)
					So(actual1, ShouldHaveLength, 4)
				})
			})
		})
	})
}

func TestErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Should throw error when no connection", t, func() {
		actual1, err := dataBase.GetAllContacts()
		So(actual1, ShouldHaveLength, 0)
		So(err, ShouldNotBeNil)

		actual2, err := dataBase.GetContact(user1Contacts[0].ID)
		So(actual2, ShouldResemble, moira.ContactData{})
		So(err, ShouldNotBeNil)

		actual3, err := dataBase.GetContacts([]string{user1Contacts[0].ID})
		So(actual3, ShouldHaveLength, 0)
		So(err, ShouldNotBeNil)

		actual4, err := dataBase.GetAllContacts()
		So(actual4, ShouldHaveLength, 0)
		So(err, ShouldNotBeNil)

		err = dataBase.SaveContact(user1Contacts[0])
		So(err, ShouldNotBeNil)

		err = dataBase.RemoveContact(user1Contacts[0].ID)
		So(err, ShouldNotBeNil)

		actual5, err := dataBase.GetUserContactIDs("123")
		So(actual5, ShouldHaveLength, 0)
		So(err, ShouldNotBeNil)
	})
}

var user1Contacts = []*moira.ContactData{
	{
		ID:    "ContactID-000000000000001",
		Type:  "email",
		Value: "mail1@example.com",
		User:  user1,
	},
	{
		ID:    "ContactID-000000000000004",
		Type:  "email",
		Value: "mail4@example.com",
		User:  user1,
	},
	{
		ID:    "ContactID-000000000000006",
		Type:  "unknown",
		Value: "no matter",
		User:  user1,
	},
	{
		ID:    "ContactID-000000000000008",
		Type:  "slack",
		Value: "#devops",
		User:  user1,
	},
}

var user2Contacts = []*moira.ContactData{
	{
		ID:    "ContactID-000000000000002",
		Type:  "email",
		Value: "failed@example.com",
		User:  user2,
	},
	{
		ID:    "ContactID-000000000000003",
		Type:  "email",
		Value: "mail3@example.com",
		User:  user2,
	},
	{
		ID:    "ContactID-000000000000005",
		Type:  "slack",
		Value: "#devops",
		User:  user2,
	},
	{
		ID:    "ContactID-000000000000007",
		Type:  "slack",
		Value: "#devops",
		User:  user2,
	},
}
