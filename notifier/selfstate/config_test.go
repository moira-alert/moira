package selfstate

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigCheck(testing *testing.T) {
	contactTypes := map[string]bool{
		"admin-mail": true,
	}

	Convey("SelfCheck disabled", testing, func() {
		config := Config{
			Enabled: false,
			Contacts: []map[string]string{
				{
					"type":  "admin-mail",
					"value": "admin@company.com",
				},
			},
		}

		Convey("all data valid, should return nil error", func() {
			actual := config.checkConfig(contactTypes)
			So(actual, ShouldBeNil)
		})

		Convey("contacts empty, should return nil error", func() {
			config.Contacts = []map[string]string{}
			actual := config.checkConfig(contactTypes)
			So(actual, ShouldBeNil)
		})

		Convey("admin sending type not registered, should return nil error", func() {
			actual := config.checkConfig(make(map[string]bool))
			So(actual, ShouldBeNil)
		})

		Convey("admin sending contact empty, should return nil error", func() {
			config.Contacts = []map[string]string{
				{
					"type":  "admin-mail",
					"value": "",
				}}
			actual := config.checkConfig(make(map[string]bool))
			So(actual, ShouldBeNil)
		})
	})

	Convey("SelfCheck contacts empty, should return contacts must be specified error", testing, func() {
		config := Config{
			Enabled: true,
		}
		actual := config.checkConfig(make(map[string]bool))
		So(actual, ShouldResemble, fmt.Errorf("contacts must be specified"))
	})

	Convey("Admin sending type not registered, should not pass check without admin contact type", testing, func() {
		config := Config{
			Enabled: true,
			Contacts: []map[string]string{
				{
					"type":  "admin-mail",
					"value": "admin@company.com",
				},
			},
		}

		actual := config.checkConfig(make(map[string]bool))
		So(actual, ShouldResemble, fmt.Errorf("unknown contact type [admin-mail]"))
	})

	Convey("Admin sending contact empty, should not pass check without admin contact", testing, func() {
		config := Config{
			Enabled: true,
			Contacts: []map[string]string{
				{
					"type":  "admin-mail",
					"value": "",
				},
			},
		}

		contactTypes := map[string]bool{
			"admin-mail": true,
		}

		actual := config.checkConfig(contactTypes)
		So(actual, ShouldResemble, fmt.Errorf("ualue for [admin-mail] must be present"))
	})

	Convey("Has registered valid admin contact, should pass check", testing, func() {
		config := Config{
			Enabled: true,
			Contacts: []map[string]string{
				{
					"type":  "admin-mail",
					"value": "admin@company.com",
				},
			},
		}

		contactTypes := map[string]bool{
			"admin-mail": true,
		}

		actual := config.checkConfig(contactTypes)
		So(actual, ShouldBeNil)
	})
}
