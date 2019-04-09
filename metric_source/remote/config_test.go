package remote

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfig(t *testing.T) {
	Convey("Given config without url and enabled", t, func(c C) {
		cfg := &Config{
			URL:     "",
			Enabled: true,
		}
		Convey("remote triggers should be disabled", t, func(c C) {
			c.So(cfg.isEnabled(), ShouldBeFalse)
		})
	})

	Convey("Given config with url and enabled", t, func(c C) {
		cfg := &Config{
			URL:     "http://host",
			Enabled: true,
		}
		Convey("remote triggers should be enabled", t, func(c C) {
			c.So(cfg.isEnabled(), ShouldBeTrue)
		})
	})

	Convey("Given config with url and disabled", t, func(c C) {
		cfg := &Config{
			URL:     "http://host",
			Enabled: false,
		}
		Convey("remote triggers should be disabled", t, func(c C) {
			c.So(cfg.isEnabled(), ShouldBeFalse)
		})
	})

	Convey("Given config without url and disabled", t, func(c C) {
		cfg := &Config{
			URL:     "",
			Enabled: false,
		}
		Convey("remote triggers should be disabled", t, func(c C) {
			c.So(cfg.isEnabled(), ShouldBeFalse)
		})
	})
}
