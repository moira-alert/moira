package reply

import (
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	testEmergencyContactVal      = `{"contact_id":"test-contact-id","emergency_types":["notifier_off"]}`
	testEmptyEmergencyContactVal = `{"contact_id":"","emergency_types":null}`
)

var (
	testEmergencyContact = moira.EmergencyContact{
		ContactID:      "test-contact-id",
		EmergencyTypes: []moira.EmergencyContactType{moira.EmergencyTypeNotifierOff},
	}
	testEmptyEmergencyContact = moira.EmergencyContact{}
)

func TestGetEmergencyContactBytes(t *testing.T) {
	Convey("Test GetEmergencyContactBytes", t, func() {
		Convey("With empty emergency contact", func() {
			emergencyContact := moira.EmergencyContact{}
			expectedEmergencyContactStr := testEmptyEmergencyContactVal
			bytes, err := GetEmergencyContactBytes(emergencyContact)
			So(err, ShouldBeNil)
			So(string(bytes), ShouldResemble, expectedEmergencyContactStr)
		})

		Convey("With test emergency contact", func() {
			expectedEmergencyContactStr := testEmergencyContactVal
			bytes, err := GetEmergencyContactBytes(testEmergencyContact)
			So(err, ShouldBeNil)
			So(string(bytes), ShouldResemble, expectedEmergencyContactStr)
		})
	})
}

func TestEmergencyContact(t *testing.T) {
	Convey("Test EmergencyContact", t, func() {
		Convey("With nil emergency contact rep", func() {
			emergencyContact, err := EmergencyContact(nil)
			So(emergencyContact, ShouldResemble, moira.EmergencyContact{})
			So(err, ShouldResemble, database.ErrNil)
		})

		Convey("With redis.Nil error in rep", func() {
			rep := &redis.StringCmd{}
			rep.SetErr(redis.Nil)
			emergencyContact, err := EmergencyContact(rep)
			So(emergencyContact, ShouldResemble, moira.EmergencyContact{})
			So(err, ShouldResemble, database.ErrNil)
		})

		Convey("With test rep", func() {
			rep := &redis.StringCmd{}
			testVal := testEmergencyContactVal
			rep.SetVal(testVal)
			emergencyContact, err := EmergencyContact(rep)
			So(emergencyContact, ShouldResemble, testEmergencyContact)
			So(err, ShouldBeNil)
		})
	})
}

func TestEmergencyContacts(t *testing.T) {
	Convey("Test EmergencyContacts", t, func() {
		Convey("With nil emergency contact rep", func() {
			emergencyContacts, err := EmergencyContacts(nil)
			So(err, ShouldBeNil)
			So(emergencyContacts, ShouldResemble, []*moira.EmergencyContact{})
		})

		Convey("With test emergency contacts rep", func() {
			rep := make([]*redis.StringCmd, 2)
			rep[0] = &redis.StringCmd{}
			rep[0].SetVal(testEmergencyContactVal)
			rep[1] = &redis.StringCmd{}
			rep[1].SetVal(testEmptyEmergencyContactVal)
			expectedEmergencyContacts := []*moira.EmergencyContact{
				&testEmergencyContact,
				&testEmptyEmergencyContact,
			}
			emergencyContacts, err := EmergencyContacts(rep)
			So(err, ShouldBeNil)
			So(emergencyContacts, ShouldResemble, expectedEmergencyContacts)
		})

		Convey("With test emergency contacts rep and one redis.Nil err", func() {
			rep := make([]*redis.StringCmd, 2)
			rep[0] = &redis.StringCmd{}
			rep[0].SetVal(testEmergencyContactVal)
			rep[1] = &redis.StringCmd{}
			rep[1].SetErr(redis.Nil)
			expectedEmergencyContacts := []*moira.EmergencyContact{
				&testEmergencyContact,
				nil,
			}
			emergencyContacts, err := EmergencyContacts(rep)
			So(err, ShouldBeNil)
			So(emergencyContacts, ShouldResemble, expectedEmergencyContacts)
		})
	})
}
