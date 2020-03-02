package moira

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnsafe(t *testing.T) {
	ShouldEqual(UnsafeBytesToString(UnsafeStringToBytes("42")), "42")
	ShouldEqual(UnsafeBytesToString(UnsafeStringToBytes("")), "")
}
