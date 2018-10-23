package moira

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

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
