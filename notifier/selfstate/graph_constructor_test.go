package selfstate

import (
	"fmt"
	"testing"

	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
	. "github.com/smartystreets/goconvey/convey"
)

func TestConstructHeartbeatsGraph(t *testing.T) {
	Convey("ConstructHeartbeatsGraph should", t, func() {
		cases := []struct {
			input    []heartbeat.Heartbeater
			expected [][]heartbeat.Heartbeater
		}{
			{
				input:    []heartbeat.Heartbeater{},
				expected: [][]heartbeat.Heartbeater(nil),
			},
			{
				input: []heartbeat.Heartbeater{
					createHeartbeat("first", false),
				},
				expected: [][]heartbeat.Heartbeater{
					{createHeartbeat("first", false)},
				},
			},
			{
				input: []heartbeat.Heartbeater{
					createHeartbeat("first", false),
					createHeartbeat("second", false),
					createHeartbeat("third", false),
				},
				expected: [][]heartbeat.Heartbeater{
					{createHeartbeat("first", false)},
					{createHeartbeat("second", false)},
					{createHeartbeat("third", false)},
				},
			},
			{
				input: []heartbeat.Heartbeater{
					createHeartbeat("first", true),
					createHeartbeat("second", true),
					createHeartbeat("third", true),
				},
				expected: [][]heartbeat.Heartbeater{
					{createHeartbeat("first", true), createHeartbeat("second", true), createHeartbeat("third", true)},
				},
			},
			{
				input: []heartbeat.Heartbeater{
					createHeartbeat("first", false),
					createHeartbeat("second", true),
					createHeartbeat("third", true),
				},
				expected: [][]heartbeat.Heartbeater{
					{createHeartbeat("first", false)},
					{createHeartbeat("second", true), createHeartbeat("third", true)},
				},
			},
			{
				input: []heartbeat.Heartbeater{
					createHeartbeat("first", false),
					createHeartbeat("second", true),
					createHeartbeat("third", true),
					createHeartbeat("fourth", false),
				},
				expected: [][]heartbeat.Heartbeater{
					{createHeartbeat("first", false)},
					{createHeartbeat("fourth", false)},
					{createHeartbeat("second", true), createHeartbeat("third", true)},
				},
			},
			{
				input: []heartbeat.Heartbeater{
					createHeartbeat("first", true),
					createHeartbeat("second", false),
					createHeartbeat("third", true),
					createHeartbeat("fourth", false),
				},
				expected: [][]heartbeat.Heartbeater{
					{createHeartbeat("second", false)},
					{createHeartbeat("fourth", false)},
					{createHeartbeat("first", true), createHeartbeat("third", true)},
				},
			},
		}

		for _, testCase := range cases {
			Convey(fmt.Sprintf("%v -> %v", testCase.input, testCase.expected), func() {
				graph := ConstructHeartbeatsGraph(testCase.input)
				So(graph, ShouldResemble, testCase.expected)
			})
		}
	})
}

func createHeartbeat(name string, needToCheckOthers bool) heartbeat.Heartbeater {
	return fakeHeartbeater{
		name:              name,
		needToCheckOthers: needToCheckOthers,
	}
}

type fakeHeartbeater struct {
	name              string
	needToCheckOthers bool
}

func (h fakeHeartbeater) Check(int64) (int64, bool, error) {
	return 0, false, nil
}

func (h fakeHeartbeater) NeedTurnOffNotifier() bool {
	return false
}

func (h fakeHeartbeater) NeedToCheckOthers() bool {
	return h.needToCheckOthers
}

func (h fakeHeartbeater) GetErrorMessage() string {
	return ""
}

func (h fakeHeartbeater) GetCheckTags() heartbeat.CheckTags {
	return []string{}
}
