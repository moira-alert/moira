package metrics

import (
	"fmt"
	"os"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInitPrefix(t *testing.T) {
	Convey("Without hostname variable, should equal to prefix", t, func(c C) {
		prefix, err := initPrefix("some.prefix.")
		c.So(prefix, ShouldEqual, prefix)
		c.So(err, ShouldBeNil)
	})
	Convey("With hostname variable, should replace", t, func(c C) {
		prefix, err := initPrefix("some.prefix.{hostname}.")
		hostname, _ := os.Hostname()
		short := strings.Split(hostname, ".")[0]
		c.So(prefix, ShouldEqual, fmt.Sprintf("some.prefix.%s.", short))
		c.So(err, ShouldBeNil)
	})
}

func TestInitRuntimeRegistry(t *testing.T) {
	runtimeRegistry := initRuntimeRegistry("service")
	Convey("Metric name should be correct", t, func(c C) {
		runtimeRegistry.Each(func(name string, i interface{}) {
			isNameCorrect := strings.HasPrefix(name, "service.runtime")
			c.So(isNameCorrect, ShouldBeTrue)
		})
	})
}
