package vmselect

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestIsConfigured(t *testing.T) {
	Convey("Remote is not configured", t, func() {
		remote := Create(&Config{URL: "", Enabled: false})
		isConfigured, _ := remote.IsConfigured()
		So(isConfigured, ShouldBeFalse)
	})

	Convey("Remote is configured", t, func() {
		remote := Create(&Config{URL: "http://host", Enabled: true})
		isConfigured, err := remote.IsConfigured()
		So(isConfigured, ShouldBeTrue)
		So(err, ShouldBeEmpty)
	})
}
