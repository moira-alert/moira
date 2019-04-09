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

	Convey("Contacts manipulation", t, func(c C) {
		Convey("While no data then get contacts should be empty", t, func(c C) {
			Convey("GetAllContacts should be empty", t, func(c C) {
				actual1, err := dataBase.GetAllContacts()
				c.So(err, ShouldBeNil)
				c.So(actual1, ShouldHaveLength, 0)
			})

			Convey("GetUserContactIDs should be empty", t, func(c C) {
				actual1, err := dataBase.GetUserContactIDs(user1)
				c.So(err, ShouldBeNil)
				c.So(actual1, ShouldHaveLength, 0)

				actual2, err := dataBase.GetUserContactIDs(user2)
				c.So(err, ShouldBeNil)
				c.So(actual2, ShouldHaveLength, 0)
			})

			Convey("GetContacts should be empty", t, func(c C) {
				actual1, err := dataBase.GetContacts([]string{user1Contacts[0].ID, user2Contacts[1].ID})
				c.So(err, ShouldBeNil)
				c.So(actual1, ShouldHaveLength, 2)
				for _, contact := range actual1 {
					c.So(contact, ShouldBeNil)
				}
			})

			Convey("GetContact should be empty", t, func(c C) {
				actual1, err := dataBase.GetContact(user1Contacts[0].ID)
				c.So(err, ShouldResemble, database.ErrNil)
				c.So(actual1, ShouldResemble, moira.ContactData{})
			})
		})

		Convey("Write all contacts for user1 and check it for success write", t, func(c C) {
			ids := make([]string, len(user1Contacts))
			for i, contact := range user1Contacts {
				ids[i] = contact.ID
				Convey(fmt.Sprintf("Write contact %s and try read", contact.ID), t, func(c C) {
					err := dataBase.SaveContact(contact)
					c.So(err, ShouldBeNil)

					actual, err := dataBase.GetContact(contact.ID)
					c.So(err, ShouldBeNil)
					c.So(actual, ShouldResemble, *contact)
				})
			}

			Convey("Read all contacts by id", t, func(c C) {
				actual, err := dataBase.GetContacts(ids)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, user1Contacts)
			})

			Convey("Read all user contacts ids", t, func(c C) {
				actual, err := dataBase.GetUserContactIDs(user1)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldHaveLength, len(ids))
			})

			Convey("Get all contacts", t, func(c C) {
				actual, err := dataBase.GetAllContacts()
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldHaveLength, len(ids))
			})
		})

		Convey("Write and remove user2 contacts by different strategies", t, func(c C) {
			ids := make([]string, len(user2Contacts))
			for i, contact := range user2Contacts {
				ids[i] = contact.ID
			}

			contact1 := user2Contacts[0]
			Convey("Save-write contact", t, func(c C) {
				Convey("Save contact", t, func(c C) {
					err := dataBase.SaveContact(contact1)
					c.So(err, ShouldBeNil)

					actual, err := dataBase.GetContact(contact1.ID)
					c.So(err, ShouldBeNil)
					c.So(actual, ShouldResemble, *contact1)

					actual1, err := dataBase.GetUserContactIDs(user2)
					c.So(err, ShouldBeNil)
					c.So(actual1, ShouldResemble, []string{contact1.ID})
				})

				Convey("Check contacts by read set of contacts", t, func(c C) {
					actual, err := dataBase.GetContacts(ids)
					c.So(err, ShouldBeNil)
					c.So(actual, ShouldHaveLength, len(ids))
					expected := make([]*moira.ContactData, len(ids))
					expected[0] = contact1
					c.So(actual, ShouldResemble, expected)
				})
			})

			contact2 := user2Contacts[1]
			Convey("Save-remove contact", t, func(c C) {
				Convey("Just save new contact", t, func(c C) {
					err := dataBase.SaveContact(contact2)
					c.So(err, ShouldBeNil)
				})

				Convey("Check it for existence", t, func(c C) {
					actual, err := dataBase.GetContact(contact2.ID)
					c.So(err, ShouldBeNil)
					c.So(actual, ShouldResemble, *contact2)

					actual1, err := dataBase.GetUserContactIDs(user2)
					c.So(err, ShouldBeNil)
					c.So(actual1, ShouldHaveLength, 2)
				})

				Convey("Remove contact", t, func(c C) {
					err := dataBase.RemoveContact(contact2.ID)
					c.So(err, ShouldBeNil)
				})

				Convey("Check it for not existence", t, func(c C) {
					actual, err := dataBase.GetContact(contact2.ID)
					c.So(err, ShouldResemble, database.ErrNil)
					c.So(actual, ShouldResemble, moira.ContactData{})

					actual1, err := dataBase.GetUserContactIDs(user2)
					c.So(err, ShouldBeNil)
					c.So(actual1, ShouldHaveLength, 1)
					c.So(actual1, ShouldResemble, []string{contact1.ID})

					actual2, err := dataBase.GetContacts(ids)
					expected := make([]*moira.ContactData, len(ids))
					expected[0] = contact1
					c.So(err, ShouldBeNil)
					c.So(actual2, ShouldResemble, expected)
				})

				Convey("And save again...", t, func(c C) {
					err := dataBase.SaveContact(contact2)
					c.So(err, ShouldBeNil)
				})

				Convey("And check it for existence again", t, func(c C) {
					actual, err := dataBase.GetContact(contact2.ID)
					c.So(err, ShouldBeNil)
					c.So(actual, ShouldResemble, *contact2)

					actual1, err := dataBase.GetUserContactIDs(user2)
					c.So(err, ShouldBeNil)
					c.So(actual1, ShouldHaveLength, 2)

					actual2, err := dataBase.GetContacts(ids)
					expected := make([]*moira.ContactData, len(ids))
					expected[0] = contact1
					expected[1] = contact2
					c.So(err, ShouldBeNil)
					c.So(actual2, ShouldResemble, expected)
				})
			})

			contact3 := *user2Contacts[2]
			contact3.User = user1
			Convey("Update contact with another user", t, func(c C) {
				Convey("Just save new contact with user1", t, func(c C) {
					err := dataBase.SaveContact(&contact3)
					c.So(err, ShouldBeNil)
				})

				Convey("Check it for existence in user1 contacts", t, func(c C) {
					actual, err := dataBase.GetContact(contact3.ID)
					c.So(err, ShouldBeNil)
					c.So(actual, ShouldResemble, contact3)

					actual1, err := dataBase.GetUserContactIDs(user2)
					c.So(err, ShouldBeNil)
					c.So(actual1, ShouldHaveLength, 2)

					actual2, err := dataBase.GetUserContactIDs(user1)
					c.So(err, ShouldBeNil)
					c.So(actual2, ShouldHaveLength, 5)
				})

				contact3.User = user2

				Convey("Now save it with user2", t, func(c C) {
					err := dataBase.SaveContact(&contact3)
					c.So(err, ShouldBeNil)
				})

				Convey("Check it for existence in user2 contacts and now existance in user1 contacts", t, func(c C) {
					actual, err := dataBase.GetContact(contact3.ID)
					c.So(err, ShouldBeNil)
					c.So(actual, ShouldResemble, contact3)

					actual1, err := dataBase.GetUserContactIDs(user2)
					c.So(err, ShouldBeNil)
					c.So(actual1, ShouldHaveLength, 3)

					actual2, err := dataBase.GetUserContactIDs(user1)
					c.So(err, ShouldBeNil)
					c.So(actual2, ShouldHaveLength, 4)
				})
			})

			contact4 := user2Contacts[3]
			Convey("Save-update contact", t, func(c C) {
				Convey("Just save new contact", t, func(c C) {
					err := dataBase.SaveContact(contact4)
					c.So(err, ShouldBeNil)
				})

				Convey("Check it for existence", t, func(c C) {
					actual, err := dataBase.GetContact(contact4.ID)
					c.So(err, ShouldBeNil)
					c.So(actual, ShouldResemble, *contact4)

					actual1, err := dataBase.GetUserContactIDs(user2)
					c.So(err, ShouldBeNil)
					c.So(actual1, ShouldHaveLength, 4)
				})

				contact2Changed := *contact4
				contact2Changed.Value = "new@email.com"

				Convey("Save updated contact data", t, func(c C) {
					err := dataBase.SaveContact(&contact2Changed)
					c.So(err, ShouldBeNil)
				})

				Convey("Check it for new data", t, func(c C) {
					actual, err := dataBase.GetContact(contact2Changed.ID)
					c.So(err, ShouldBeNil)
					c.So(actual, ShouldResemble, contact2Changed)

					actual1, err := dataBase.GetUserContactIDs(user2)
					c.So(err, ShouldBeNil)
					c.So(actual1, ShouldHaveLength, 4)
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

	Convey("Should throw error when no connection", t, func(c C) {
		actual1, err := dataBase.GetAllContacts()
		c.So(actual1, ShouldHaveLength, 0)
		c.So(err, ShouldNotBeNil)

		actual2, err := dataBase.GetContact(user1Contacts[0].ID)
		c.So(actual2, ShouldResemble, moira.ContactData{})
		c.So(err, ShouldNotBeNil)

		actual3, err := dataBase.GetContacts([]string{user1Contacts[0].ID})
		c.So(actual3, ShouldHaveLength, 0)
		c.So(err, ShouldNotBeNil)

		actual4, err := dataBase.GetAllContacts()
		c.So(actual4, ShouldHaveLength, 0)
		c.So(err, ShouldNotBeNil)

		err = dataBase.SaveContact(user1Contacts[0])
		c.So(err, ShouldNotBeNil)

		err = dataBase.RemoveContact(user1Contacts[0].ID)
		c.So(err, ShouldNotBeNil)

		actual5, err := dataBase.GetUserContactIDs("123")
		c.So(actual5, ShouldHaveLength, 0)
		c.So(err, ShouldNotBeNil)
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
