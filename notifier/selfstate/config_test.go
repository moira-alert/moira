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

	Convey("SelfCheck disabled", testing, func(c C) {
		config := Config{
			Enabled: false,
			Contacts: []map[string]string{
				{
					"type":  "admin-mail",
					"value": "admin@company.com",
				},
			},
		}

		Convey("all data valid, should return nil error", t, func(c C) {
			actual := config.checkConfig(contactTypes)
			c.So(actual, ShouldBeNil)
		})

		Convey("contacts empty, should return nil error", t, func(c C) {
			config.Contacts = []map[string]string{}
			actual := config.checkConfig(contactTypes)
			c.So(actual, ShouldBeNil)
		})

		Convey("admin sending type not registered, should return nil error", t, func(c C) {
			actual := config.checkConfig(make(map[string]bool))
			c.So(actual, ShouldBeNil)
		})

		Convey("admin sending contact empty, should return nil error", t, func(c C) {
			config.Contacts = []map[string]string{
				{
					"type":  "admin-mail",
					"value": "",
				}}
			actual := config.checkConfig(make(map[string]bool))
			c.So(actual, ShouldBeNil)
		})
	})

	Convey("SelfCheck contacts empty, should return contacts must be specified error", testing, func(c C) {
		config := Config{
			Enabled: true,
		}
		actual := config.checkConfig(make(map[string]bool))
		c.So(actual, ShouldResemble, fmt.Errorf("contacts must be specified"))
	})

	Convey("Admin sending type not registered, should not pass check without admin contact type", testing, func(c C) {
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
		c.So(actual, ShouldResemble, fmt.Errorf("unknown contact type [admin-mail]"))
	})

	Convey("Admin sending contact empty, should not pass check without admin contact", testing, func(c C) {
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
		c.So(actual, ShouldResemble, fmt.Errorf("value for [admin-mail] must be present"))
	})

	Convey("Has registered valid admin contact, should pass check", testing, func(c C) {
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
		c.So(actual, ShouldBeNil)
	})
}
