package moira

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"slices"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBytesScanner(t *testing.T) {
	type BytesScannerTestCase struct {
		input  string
		output []string
	}

	Convey("", t, func() {
		cases := []BytesScannerTestCase{
			{input: "", output: []string{}},
			{input: "a", output: []string{"a"}},
			{input: " ", output: []string{"", ""}},
			{input: "a ", output: []string{"a", ""}},
			{input: " a", output: []string{"", "a"}},
			{input: " a ", output: []string{"", "a", ""}},
			{input: "a a", output: []string{"a", "a"}},
		}
		for _, c := range cases {
			actualOutput := make([]string, 0)
			scanner := NewBytesScanner([]byte(c.input), ' ')

			for scanner.HasNext() {
				actualOutput = append(actualOutput, string(scanner.Next()))
			}

			So(actualOutput, ShouldResemble, c.output)
		}
	})
}

func TestInt64ToTime(t *testing.T) {
	int64timeStamp := int64(1527330278)
	humanReadableTimestamp := time.Date(2018, 5, 26, 10, 24, 38, 0, time.UTC)

	Convey("Convert int64 timestamp into datetime", t, func() {
		converted := Int64ToTime(int64timeStamp)
		So(converted, ShouldResemble, humanReadableTimestamp)
	})
	Convey("Convert int64 timestamp + 1 minute into datetime", t, func() {
		int64timeStamp += 60
		converted := Int64ToTime(int64timeStamp)
		So(converted, ShouldResemble, humanReadableTimestamp.Add(time.Minute))
	})
}

func TestSubset(t *testing.T) {
	Convey("Test subsets", t, func() {
		So(Subset([]string{"1", "2", "3"}, []string{"3", "2", "1"}), ShouldBeTrue)
		So(Subset([]string{"1", "2", "3"}, []string{"1", "1", "1", "2", "2", "2", "3", "3", "3"}), ShouldBeTrue)
		So(Subset([]string{"1", "2", "3"}, []string{"123", "2", "3"}), ShouldBeFalse)
		So(Subset([]string{"1", "2", "3"}, []string{"1", "2", "4"}), ShouldBeFalse)
	})
}

func TestGetStringListsDiff(t *testing.T) {
	Convey("Test Get Difference between string lists", t, func() {
		first := []string{"1", "2", "3"}
		second := []string{"1", "2", "3"}
		diff := GetStringListsDiff(first, second)
		So(diff, ShouldResemble, []string{})

		first = []string{"1", "2", "3", "4", "5"}
		diff = GetStringListsDiff(first, second)
		So(diff, ShouldResemble, []string{"4", "5"})

		second = []string{"6", "7", "8"}
		diff = GetStringListsDiff(first, second)
		So(diff, ShouldResemble, first)

		third := []string{"8", "9", "10"}
		diff = GetStringListsDiff(first, second, third)
		So(diff, ShouldResemble, first)

		first = []string{"1", "2", "3", "4", "5", "6", "7", "8"}
		second = []string{"6", "7", "8"}
		third = []string{"8", "9", "10"}
		diff = GetStringListsDiff(first, second, third)
		So(diff, ShouldResemble, []string{"1", "2", "3", "4", "5"})
	})
}

func TestGetUniqueValues(t *testing.T) {
	Convey("Test Get Unique Values of list", t, func() {
		cases := []struct {
			input    []string
			expected []string
		}{
			{
				input:    []string{},
				expected: []string{},
			},
			{
				input:    []string{"a"},
				expected: []string{"a"},
			},
			{
				input:    []string{"a", "b"},
				expected: []string{"a", "b"},
			},
			{
				input:    []string{"a", "a"},
				expected: []string{"a"},
			},
			{
				input:    []string{"a", "a", "b", "b", "c", "c"},
				expected: []string{"a", "b", "c"},
			},
		}
		for _, variant := range cases {
			Convey(fmt.Sprintf("with %v -> %v", variant.input, variant.expected), func() {
				actual := GetUniqueValues(variant.input...)
				slices.Sort(actual)
				So(actual, ShouldResemble, variant.expected)
			})
		}
	})
}

func TestIntersect(t *testing.T) {
	Convey("Test Intersect lists", t, func() {
		cases := []struct {
			input    [][]string
			expected []string
		}{
			{
				input:    [][]string{{}},
				expected: []string{},
			},
			{
				input:    [][]string{{}, {}},
				expected: []string{},
			},
			{
				input:    [][]string{{"a"}},
				expected: []string{"a"},
			},
			{
				input:    [][]string{{"a", "b"}, {"a", "c"}},
				expected: []string{"a"},
			},
			{
				input:    [][]string{{"a", "b", "c", "d"}, {"e", "f", "g"}},
				expected: []string{},
			},
			{
				input:    [][]string{{"a", "b", "e", "d"}, {"12", "f", "e"}},
				expected: []string{"e"},
			},
			{
				input:    [][]string{{"a"}, {}},
				expected: []string{},
			},
		}
		for _, variant := range cases {
			Convey(fmt.Sprintf("intersect(%v) -> %v", variant.input, variant.expected), func() {
				actual := Intersect(variant.input...)
				slices.Sort(actual)
				So(actual, ShouldResemble, variant.expected)
			})
		}
	})
}

func TestGetTriggerListsDiff(t *testing.T) {
	Convey("Test Get Difference between trigger lists", t, func() {
		first := []*Trigger{triggerVal1, triggerVal2}
		second := []*Trigger{triggerVal3, triggerVal1, triggerVal2, triggerVal4}
		diff := GetTriggerListsDiff(first, second)
		So(diff, ShouldResemble, []*Trigger{})

		first = []*Trigger{triggerVal1, triggerVal2, triggerVal3, triggerVal4}
		second = []*Trigger{triggerVal2, triggerVal2}
		third := []*Trigger{triggerVal3, triggerVal3, triggerVal3}
		diff = GetTriggerListsDiff(first, second, third)
		So(diff, ShouldResemble, []*Trigger{triggerVal1, triggerVal4})

		Convey("One trigger in first array is nil", func() {
			first = []*Trigger{nil, triggerVal2, triggerVal3, triggerVal4}
			second = []*Trigger{triggerVal2}
			diff = GetTriggerListsDiff(first, second)
			So(diff, ShouldResemble, []*Trigger{triggerVal3, triggerVal4})
		})

		Convey("One trigger in additional arrays in nil", func() {
			first = []*Trigger{triggerVal1, triggerVal2, triggerVal3, triggerVal4}
			second = []*Trigger{nil}
			diff = GetTriggerListsDiff(first, second)
			So(diff, ShouldResemble, []*Trigger{triggerVal1, triggerVal2, triggerVal3, triggerVal4})
		})

		Convey("First array is empty", func() {
			first = []*Trigger{nil, nil, nil}
			second = []*Trigger{triggerVal1}
			diff = GetTriggerListsDiff(first, second)
			So(diff, ShouldResemble, []*Trigger{})
		})

		Convey("Additional arrays is empty", func() {
			first = []*Trigger{triggerVal1}
			second = []*Trigger{nil}
			third = []*Trigger{nil}
			diff = GetTriggerListsDiff(first, second, third)
			So(diff, ShouldResemble, first)
		})
	})
}

var triggerVal1 = &Trigger{
	ID:   "trigger-id-1",
	Name: "Super Trigger 1",
	Tags: []string{"test", "super", "1"},
}

var triggerVal2 = &Trigger{
	ID:   "trigger-id-2",
	Name: "Super Trigger 2",
	Tags: []string{"test", "2"},
}

var triggerVal3 = &Trigger{
	ID:   "trigger-id-3",
	Name: "Super Trigger 3",
	Tags: []string{"super", "3"},
}

var triggerVal4 = &Trigger{
	ID:            "trigger-id-4",
	Name:          "Super Trigger 4",
	TriggerSource: GraphiteRemote,
	TTL:           600,
	Tags:          []string{"4"},
}

func TestChunkSlice(t *testing.T) {
	Convey("Test chunking slices", t, func() {
		originalSlice := []string{"123", "234", "345", "456", "567", "678", "789", "890"}

		actual := ChunkSlice(originalSlice, 10)
		So(actual, ShouldResemble, [][]string{originalSlice})

		actual = ChunkSlice(originalSlice, 1)
		So(actual, ShouldResemble, [][]string{{"123"}, {"234"}, {"345"}, {"456"}, {"567"}, {"678"}, {"789"}, {"890"}})

		actual = ChunkSlice(originalSlice, 5)
		So(actual, ShouldResemble, [][]string{{"123", "234", "345", "456", "567"}, {"678", "789", "890"}})

		actual = ChunkSlice(originalSlice, 0)
		So(actual, ShouldBeEmpty)
	})
}

func TestIsValidFloat64(t *testing.T) {
	Convey("values +Inf -Inf and NaN is invalid", t, func() {
		So(IsFiniteNumber(math.NaN()), ShouldBeFalse)
		So(IsFiniteNumber(math.Inf(-1)), ShouldBeFalse)
		So(IsFiniteNumber(math.Inf(1)), ShouldBeFalse)
		So(IsFiniteNumber(3.14), ShouldBeTrue)
	})
}

func TestRoundToNearestRetention(t *testing.T) {
	var baseTimestamp, retentionSeconds int64
	baseTimestamp = 1582286400 // 17:00:00

	Convey("Test even retention: 60s", t, func() {
		retentionSeconds = 60
		testRounding(baseTimestamp, retentionSeconds)
	})

	Convey("Test odd retention: 15s", t, func() {
		retentionSeconds = 15
		testRounding(baseTimestamp, retentionSeconds)
	})
}

func testRounding(baseTimestamp, retention int64) {
	halfRetention := retention / 2
	nextTimestamp := baseTimestamp + retention

	Convey("round to self", func() {
		So(RoundToNearestRetention(baseTimestamp, retention), ShouldEqual, baseTimestamp)
	})

	So(RoundToNearestRetention(baseTimestamp+1, retention), ShouldEqual, baseTimestamp)
	So(RoundToNearestRetention(baseTimestamp+(halfRetention-1), retention), ShouldEqual, baseTimestamp)
	So(RoundToNearestRetention(baseTimestamp+halfRetention+retention%2, retention), ShouldEqual, nextTimestamp)

	So(RoundToNearestRetention(baseTimestamp-1, retention), ShouldEqual, baseTimestamp)
	So(RoundToNearestRetention(baseTimestamp-halfRetention, retention), ShouldEqual, baseTimestamp)
}

func TestReplaceSubstring(t *testing.T) {
	Convey("Test replace substring", t, func() {
		Convey("replacement string in the middle", func() {
			So(ReplaceSubstring("telebot: Post https://api.telegram.org/botXXX/getMe", "/bot", "/", "[DELETED]"), ShouldResemble, "telebot: Post https://api.telegram.org/bot[DELETED]/getMe")
		})

		Convey("replacement string at the beginning", func() {
			So(ReplaceSubstring("/botXXX/getMe", "/bot", "/", "[DELETED]"), ShouldResemble, "/bot[DELETED]/getMe")
		})

		Convey("replacement string at the end", func() {
			So(ReplaceSubstring("telebot: Post https://api.telegram.org/botXXX/", "/bot", "/", "[DELETED]"), ShouldResemble, "telebot: Post https://api.telegram.org/bot[DELETED]/")
		})

		Convey("no replacement string", func() {
			So(ReplaceSubstring("telebot: Post https://api.telegram.org/getMe", "/bot", "/", "[DELETED]"), ShouldResemble, "telebot: Post https://api.telegram.org/getMe")
		})

		Convey("there is the beginning of replacement string, but no end", func() {
			So(ReplaceSubstring("https://api.telegram.org/botXXX error", "/bot", "/", "[DELETED]"), ShouldResemble, "https://api.telegram.org/botXXX error")
		})
	})
}

type myInt int

func (m myInt) Less(other Comparable) (bool, error) {
	otherInt := other.(myInt)
	return m < otherInt, nil
}

type myTest struct {
	value int
}

func (test myTest) Less(other Comparable) (bool, error) {
	otherTest := other.(myTest)
	return test.value < otherTest.value, nil
}

func TestMergeToSorted(t *testing.T) {
	Convey("Test MergeToSorted function", t, func() {
		Convey("Test with two nil arrays", func() {
			merged, err := MergeToSorted[myInt](nil, nil)
			So(err, ShouldBeNil)
			So(merged, ShouldResemble, []myInt{})
		})

		Convey("Test with one nil array", func() {
			merged, err := MergeToSorted(nil, []myInt{1, 2, 3})
			So(err, ShouldBeNil)
			So(merged, ShouldResemble, []myInt{1, 2, 3})
		})

		Convey("Test with two arrays", func() {
			merged, err := MergeToSorted([]myInt{4, 5}, []myInt{1, 2, 3})
			So(err, ShouldBeNil)
			So(merged, ShouldResemble, []myInt{1, 2, 3, 4, 5})
		})

		Convey("Test with empty array", func() {
			merged, err := MergeToSorted([]myInt{-4, 5}, []myInt{})
			So(err, ShouldBeNil)
			So(merged, ShouldResemble, []myInt{-4, 5})
		})

		Convey("Test with sorted values but mixed up", func() {
			merged, err := MergeToSorted([]myInt{1, 9, 10}, []myInt{4, 8, 12})
			So(err, ShouldBeNil)
			So(merged, ShouldResemble, []myInt{1, 4, 8, 9, 10, 12})
		})

		Convey("Test with structure type", func() {
			arr1 := []myTest{
				{
					value: 1,
				},
				{
					value: 2,
				},
			}

			arr2 := []myTest{
				{
					value: -2,
				},
				{
					value: -1,
				},
			}

			expected := append(arr2, arr1...)
			merged, err := MergeToSorted(arr1, arr2)
			So(err, ShouldBeNil)
			So(merged, ShouldResemble, expected)
		})
	})
}

func TestValidateStruct(t *testing.T) {
	type ValidationStruct struct {
		TestInt  int    `validate:"required,gt=0"`
		TestURL  string `validate:"required,url"`
		TestBool bool
	}

	const (
		validURL = "https://github.com/moira-alert/moira"
		validInt = 1
	)

	Convey("Test ValidateStruct", t, func() {
		Convey("With TestInt less than zero", func() {
			testStruct := ValidationStruct{
				TestInt: -1,
				TestURL: validURL,
			}

			err := ValidateStruct(testStruct)
			So(err, ShouldNotBeNil)
		})

		Convey("With invalid TestURL format", func() {
			testStruct := ValidationStruct{
				TestInt:  validInt,
				TestURL:  "test",
				TestBool: true,
			}

			err := ValidateStruct(testStruct)
			So(err, ShouldNotBeNil)
		})

		Convey("With valid structure", func() {
			testStruct := ValidationStruct{
				TestInt: validInt,
				TestURL: validURL,
			}

			err := ValidateStruct(testStruct)
			So(err, ShouldBeNil)
		})
	})
}

func TestValidateURL(t *testing.T) {
	Convey("Test ValidateURL", t, func() {
		type testcase struct {
			desc        string
			givenURL    string
			expectedErr error
		}

		cases := []testcase{
			{
				desc:     "no scheme",
				givenURL: "hello.example.com/path",
				expectedErr: &url.Error{
					Op:  "parse",
					URL: "hello.example.com/path",
					Err: errors.New("invalid URI for request"),
				},
			},
			{
				desc:        "valid url",
				givenURL:    "http://example.com/path?query=1&oksome=2",
				expectedErr: nil,
			},
			{
				desc:        "bad scheme",
				givenURL:    "smtp://example.com/path/to?query=1&oksome=2",
				expectedErr: fmt.Errorf("bad url scheme: %s", "smtp"),
			},
			{
				desc:        "no host",
				givenURL:    "https:///path/to?query=1&some=2",
				expectedErr: fmt.Errorf("host is empty"),
			},
		}

		for i, singleCase := range cases {
			Convey(fmt.Sprintf("Case %v: %s", i+1, singleCase.desc), func() {
				err := ValidateURL(singleCase.givenURL)
				So(err, ShouldResemble, singleCase.expectedErr)
			})
		}
	})
}
