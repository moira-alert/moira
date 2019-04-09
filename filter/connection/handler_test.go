package connection

import "testing"
import . "github.com/smartystreets/goconvey/convey"

func TestDropCRLF(t *testing.T) {
	type TestCase struct {
		input  []byte
		output []byte
	}

	Convey("Should drop CRLF", t, func(c C) {
		testCases := []TestCase{
			{[]byte{}, []byte{}},
			{[]byte{'a'}, []byte{'a'}},
			{[]byte{'\n'}, []byte{}},
			{[]byte{'\r'}, []byte{}},
			{[]byte{'\r', '\n'}, []byte{}},
		}

		for _, testCase := range testCases {
			output := dropCRLF(testCase.input)
			c.So(testCase.output, ShouldResemble, output)
		}
	})
}
