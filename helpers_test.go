package moira

import (
	"math"
	"sort"
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

func TestGetStringListsUnion(t *testing.T) {
	Convey("Test Get Union between string lists", t, func() {
		{
			union := GetStringListsUnion(nil, nil)
			So(union, ShouldResemble, []string{})
		}
		{
			first := []string{"1", "2", "3"}
			second := []string{"1", "2", "3"}
			union := GetStringListsUnion(first, second)
			sort.Strings(union)
			So(union, ShouldResemble, []string{"1", "2", "3"})
		}
		{
			first := []string{"1", "2", "3"}
			second := []string{"4", "5", "6"}
			union := GetStringListsUnion(first, second)
			sort.Strings(union)
			So(union, ShouldResemble, []string{"1", "2", "3", "4", "5", "6"})
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
	ID:       "trigger-id-4",
	Name:     "Super Trigger 4",
	IsRemote: true,
	TTL:      600,
	Tags:     []string{"4"},
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
		So(IsValidFloat64(math.NaN()), ShouldBeFalse)
		So(IsValidFloat64(math.Inf(-1)), ShouldBeFalse)
		So(IsValidFloat64(math.Inf(1)), ShouldBeFalse)
		So(IsValidFloat64(3.14), ShouldBeTrue)
	})
}
