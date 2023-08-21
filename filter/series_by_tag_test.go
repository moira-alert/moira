package filter

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCleanAsterisks(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"*", ".*"},
		{"**", ".*.*"},
		{"aword.*", "aword.*"},
		{"aword*", "aword.*"},
		{"*aword", ".*aword"},
		{".*aword", ".*aword"},
		{"^aword$", "^aword$"},
		{"*lor*", ".*lor.*"},
		{"aw.*d*or", "aw.*d.*or"},
	}
	Convey("testing clean asterisks", t, func() {
		for _, tc := range testcases {
			output := cleanAsterisks(tc.input)
			So(output, ShouldResemble, tc.expected)
		}
	})
}
