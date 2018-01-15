package metrics

import (
	"fmt"
	"os"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInitPrefix(t *testing.T) {
	Convey("Without hostname variable, should equal to prefix", t, func() {
		prefix, err := initPrefix("some.prefix.")
		So(prefix, ShouldEqual, prefix)
		So(err, ShouldBeNil)
	})
	Convey("With hostname variable, should replace", t, func() {
		prefix, err := initPrefix("some.prefix.{hostname}.")
		hostname, _ := os.Hostname()
		short := strings.Split(hostname, ".")[0]
		So(prefix, ShouldEqual, fmt.Sprintf("some.prefix.%s.", short))
		So(err, ShouldBeNil)
	})
}
