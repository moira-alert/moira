package dto

import (
	"testing"

	"github.com/moira-alert/moira/datatypes"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testContactID = "test-contact-id"

	testEmergencyContact = datatypes.EmergencyContact{
		ContactID:      testContactID,
		HeartbeatTypes: []datatypes.HeartbeatType{datatypes.HeartbeatNotifierOff},
	}
)

func TestFromEmergencyContacts(t *testing.T) {
	Convey("Test FromEmergencyContacts", t, func() {
		Convey("With nil emergency contacts", func() {
			expectedEmergencyContactList := &EmergencyContactList{
				List: make([]EmergencyContact, 0),
			}
			emergencyContactList := FromEmergencyContacts(nil)
			So(emergencyContactList, ShouldResemble, expectedEmergencyContactList)
		})

		Convey("With some emergency contacts", func() {
			expectedEmergencyContactList := &EmergencyContactList{
				List: []EmergencyContact{
					EmergencyContact(testEmergencyContact),
				},
			}
			emergencyContacts := []*datatypes.EmergencyContact{
				&testEmergencyContact,
				nil,
			}
			emergencyContactList := FromEmergencyContacts(emergencyContacts)
			So(emergencyContactList, ShouldResemble, expectedEmergencyContactList)
		})
	})
}
