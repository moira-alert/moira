package redis

import (
	"errors"
	"testing"

	"github.com/moira-alert/moira"
	moiradb "github.com/moira-alert/moira/database"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

var (
	testContactID  = "test-contact-id"
	testContactID2 = "test-contact-id2"
	testContactID3 = "test-contact-id3"

	testEmergencyContact = moira.EmergencyContact{
		ContactID:      testContactID,
		HeartbeatTypes: []moira.HeartbeatType{moira.HeartbeatNotifierOff},
	}

	testEmergencyContact2 = moira.EmergencyContact{
		ContactID:      testContactID2,
		HeartbeatTypes: []moira.HeartbeatType{moira.HeartbeatNotifierOff},
	}

	testEmergencyContact3 = moira.EmergencyContact{
		ContactID:      testContactID3,
		HeartbeatTypes: []moira.HeartbeatType{moira.HearbeatTypeNotSet},
	}
)

func TestGetEmergencyContact(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()

	Convey("Test GetEmergencyContact", t, func() {
		Convey("With unknown emergency contact", func() {
			emergencyContact, err := database.GetEmergencyContact(testContactID)
			So(err, ShouldResemble, moiradb.ErrNil)
			So(emergencyContact, ShouldResemble, moira.EmergencyContact{})
		})

		Convey("With some emergency contact", func() {
			err := database.SaveEmergencyContact(testEmergencyContact)
			So(err, ShouldBeNil)

			emergencyContact, err := database.GetEmergencyContact(testContactID)
			So(err, ShouldBeNil)
			So(emergencyContact, ShouldResemble, testEmergencyContact)
		})
	})
}

func TestGetEmergencyContacts(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()

	Convey("Test GetEmergencyContacts", t, func() {
		Convey("Without emergency contacts", func() {
			emergencyContacts, err := database.GetEmergencyContacts()
			So(err, ShouldBeNil)
			So(emergencyContacts, ShouldResemble, []*moira.EmergencyContact{})
		})

		Convey("With some emergency contacts", func() {
			database.saveEmergencyContacts([]moira.EmergencyContact{
				testEmergencyContact,
				testEmergencyContact2,
				testEmergencyContact3,
			})

			expectedEmergencyContacts := []*moira.EmergencyContact{
				&testEmergencyContact,
				&testEmergencyContact2,
				&testEmergencyContact3,
			}

			emergencyContacts, err := database.GetEmergencyContacts()
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, emergencyContacts, expectedEmergencyContacts)
		})
	})
}

func TestGetEmergencyContactsByIDs(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()

	Convey("Test GetEmergencyContactsByIDs", t, func() {
		Convey("With empty contact ids", func() {
			contactIDs := []string{}
			emergencyContacts, err := database.GetEmergencyContactsByIDs(contactIDs)
			So(err, ShouldBeNil)
			So(emergencyContacts, ShouldResemble, []*moira.EmergencyContact{})
		})

		Convey("With some saved contact ids", func() {
			database.saveEmergencyContacts([]moira.EmergencyContact{
				testEmergencyContact,
				testEmergencyContact2,
			})

			expectedEmergencyContacts := []*moira.EmergencyContact{
				&testEmergencyContact,
				&testEmergencyContact2,
			}

			contactIDs := []string{testContactID, testContactID2}
			emergencyContacts, err := database.GetEmergencyContactsByIDs(contactIDs)
			So(err, ShouldBeNil)
			So(emergencyContacts, ShouldResemble, expectedEmergencyContacts)
		})

		Convey("With one saved and one not saved contact ids", func() {
			database.Flush()
			database.saveEmergencyContacts([]moira.EmergencyContact{
				testEmergencyContact,
			})

			expectedEmergencyContacts := []*moira.EmergencyContact{
				&testEmergencyContact,
				nil,
			}

			contactIDs := []string{testContactID, testContactID2}
			emergencyContacts, err := database.GetEmergencyContactsByIDs(contactIDs)
			So(err, ShouldBeNil)
			So(emergencyContacts, ShouldResemble, expectedEmergencyContacts)
		})
	})
}

func TestGetEmergencyTypeContactIDs(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()

	Convey("Test GetHeartbeatTypeContactIDs", t, func() {
		Convey("Without any emergency contacts by heartbeat type", func() {
			emergencyContactIDs, err := database.GetHeartbeatTypeContactIDs(moira.HeartbeatNotifierOff)
			So(err, ShouldBeNil)
			So(emergencyContactIDs, ShouldBeEmpty)
		})

		Convey("With some emergency contacts by type", func() {
			database.saveEmergencyContacts([]moira.EmergencyContact{
				testEmergencyContact,
				testEmergencyContact2,
				testEmergencyContact3,
			})

			emergencyContactIDs, err := database.GetHeartbeatTypeContactIDs(moira.HeartbeatNotifierOff)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, emergencyContactIDs, []string{
				testContactID,
				testContactID2,
			})

			emergencyContactIDs, err = database.GetHeartbeatTypeContactIDs(moira.HeartbeatNotifierOff)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, emergencyContactIDs, []string{
				testContactID3,
			})
		})
	})
}

func TestSaveEmergencyContact(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()

	Convey("Test SaveEmergencyContact", t, func() {
		Convey("With some emergency contact", func() {
			expectedEmergencyContacts := []*moira.EmergencyContact{&testEmergencyContact}
			expectedEmergencyContactIDs := []string{testContactID}

			emergencyContacts, err := database.GetEmergencyContacts()
			So(err, ShouldBeNil)
			So(emergencyContacts, ShouldBeEmpty)

			err = database.SaveEmergencyContact(testEmergencyContact)
			So(err, ShouldBeNil)

			emergencyContacts, err = database.GetEmergencyContacts()
			So(err, ShouldBeNil)
			So(emergencyContacts, ShouldResemble, expectedEmergencyContacts)

			emergencyContactIDs, err := database.GetHeartbeatTypeContactIDs(moira.HeartbeatNotifierOff)
			So(err, ShouldBeNil)
			So(emergencyContactIDs, ShouldResemble, expectedEmergencyContactIDs)
		})
	})
}

func TestSaveEmergencyContacts(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()

	Convey("Test saveEmergencyContacts", t, func() {
		Convey("With some emergency contacts", func() {
			expectedEmergencyContacts := []*moira.EmergencyContact{&testEmergencyContact, &testEmergencyContact2, &testEmergencyContact3}
			expectedEmergencyContactIDs := []string{testContactID, testContactID2}

			emergencyContacts, err := database.GetEmergencyContacts()
			So(err, ShouldBeNil)
			So(emergencyContacts, ShouldBeEmpty)

			err = database.saveEmergencyContacts([]moira.EmergencyContact{
				testEmergencyContact,
				testEmergencyContact2,
				testEmergencyContact3,
			})
			So(err, ShouldBeNil)

			emergencyContacts, err = database.GetEmergencyContacts()
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, emergencyContacts, expectedEmergencyContacts)

			emergencyContactIDs, err := database.GetHeartbeatTypeContactIDs(moira.HeartbeatNotifierOff)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, emergencyContactIDs, expectedEmergencyContactIDs)
		})
	})
}

func TestRemoveEmergencyContact(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()

	Convey("Test RemoveEmergencyContact", t, func() {
		Convey("With unknown emergency contact", func() {
			err := database.RemoveEmergencyContact(testContactID)
			So(errors.Is(err, moiradb.ErrNil), ShouldBeTrue)
		})

		Convey("With some emergency contact", func() {
			err := database.SaveEmergencyContact(testEmergencyContact)
			So(err, ShouldBeNil)

			emergencyContact, err := database.GetEmergencyContact(testContactID)
			So(err, ShouldBeNil)
			So(emergencyContact, ShouldResemble, testEmergencyContact)

			err = database.RemoveEmergencyContact(testContactID)
			So(err, ShouldResemble, nil)

			emergencyContact, err = database.GetEmergencyContact(testContactID)
			So(errors.Is(err, moiradb.ErrNil), ShouldBeTrue)
			So(emergencyContact, ShouldResemble, moira.EmergencyContact{})
		})
	})
}
