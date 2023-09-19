package prometheus

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestIsConfigured(t *testing.T) {
	Convey("Metric source is not configured", t, func() {
		source, _ := Create(&Config{URL: "", Enabled: false}, nil)
		isConfigured, err := source.IsConfigured()
		So(isConfigured, ShouldBeFalse)
		So(err, ShouldResemble, ErrPrometheusStorageDisabled)
	})

	Convey("Metric source is configured", t, func() {
		source, _ := Create(&Config{URL: "http://host", Enabled: true}, nil)
		isConfigured, err := source.IsConfigured()
		So(isConfigured, ShouldBeTrue)
		So(err, ShouldBeEmpty)
	})
}
