package selfstate

import (
	"fmt"
	"testing"

	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMergeLayerResults(t *testing.T) {
	Convey("MergeLayerResults shoud return expected", t, func() {
		testCases := []struct {
			desc     string
			input    []heartbeaterCheckResult
			expected graphExecutionResult
		}{
			{
				desc:     "if empty layer results",
				input:    []heartbeaterCheckResult{},
				expected: graphExecutionResult{0, 0, false, false, nil, nil},
			},
			{
				desc: "if single layer result",
				input: []heartbeaterCheckResult{
					{0, false, nil, false, "", []string{}},
				},
				expected: graphExecutionResult{0, 0, false, false, nil, nil},
			},
			{
				desc: "if all success",
				input: []heartbeaterCheckResult{
					{0, false, nil, false, "", []string{}},
					{0, false, nil, false, "", []string{}},
					{0, false, nil, false, "", []string{}},
				},
				expected: graphExecutionResult{0, 0, false, false, nil, nil},
			},
			{
				desc: "if single error",
				input: []heartbeaterCheckResult{
					{0, false, nil, false, "", []string{}},
					{10, true, nil, false, "some error", []string{}},
					{0, false, nil, false, "", []string{}},
				},
				expected: graphExecutionResult{10, 0, true, false, []string{"some error"}, nil},
			},
			{
				desc: "if multiple errors",
				input: []heartbeaterCheckResult{
					{10, true, nil, false, "first error", []string{}},
					{15, true, nil, false, "second error", []string{}},
					{0, false, nil, false, "", []string{}},
				},
				expected: graphExecutionResult{15, 0, true, false, []string{"first error", "second error"}, nil}
			},
			{
				desc: "if all are errors",
				input: []heartbeaterCheckResult{
					{10, true, nil, false, "first error", []string{}},
					{11, true, nil, true, "second error", []string{}},
					{12, true, nil, false, "third error", []string{}},
				},
				expected: graphExecutionResult{12, 0, true, true, []string{"first error", "second error", "third error"}, nil},
			},
		}

		for _, testCase := range testCases {
			Convey(fmt.Sprintf("%v", testCase.desc), func() {
				actual, err := mergeLayerResults(testCase.input...)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, testCase.expected)
			})
		}
	})
}

func TestExecuteGraph(t *testing.T) {
	Convey("ExecuteGraph should return expected", t, func() {
		testCases := []struct {
			desc     string
			input    heartbeatsGraph
			expected graphExecutionResult
		}{
			{
				desc:     "if graph is empty",
				input:    [][]heartbeat.Heartbeater{},
				expected: graphExecutionResult{0, 0, false, false, nil, nil},
			},
			{
				desc:     "if graph contains one empty layer",
				input:    [][]heartbeat.Heartbeater{{}},
				expected: graphExecutionResult{0, 0, false, false, nil, nil},
			},
			{
				desc: "if graph contains one check",
				input: [][]heartbeat.Heartbeater{
					{simpleHeartbeater{heartbeaterCheckResult{0, false, nil, false, "", []string{}}}},
				},
				expected: graphExecutionResult{0, 0, false, false, nil, nil},
			},
			{
				desc: "if graph contains multiple checks on same layer",
				input: [][]heartbeat.Heartbeater{
					{
						simpleHeartbeater{heartbeaterCheckResult{0, false, nil, false, "", []string{}}},
						simpleHeartbeater{heartbeaterCheckResult{0, false, nil, false, "", []string{}}},
						simpleHeartbeater{heartbeaterCheckResult{0, false, nil, false, "", []string{}}},
					},
				},
				expected: graphExecutionResult{0, 0, false, false, nil, nil},
			},
			{
				desc: "if graph contains one failed check",
				input: [][]heartbeat.Heartbeater{
					{simpleHeartbeater{heartbeaterCheckResult{10, true, nil, false, "some error", []string{"tag"}}}},
				},
				expected: graphExecutionResult{10, 0, true, false, []string{"some error"}, []string{"tag"}},
			},
			{
				desc: "if graph contains multiple failed checks on same layer",
				input: [][]heartbeat.Heartbeater{
					{
						simpleHeartbeater{heartbeaterCheckResult{10, true, nil, false, "some error", []string{"tag"}}},
						simpleHeartbeater{heartbeaterCheckResult{15, true, nil, false, "some error", []string{"tag"}}},
					},
				},
				expected: graphExecutionResult{15, 0, true, false, []string{"some error", "some error"}, []string{"tag", "tag"}},
			},
			{
				desc: "if graph contains multiple failed checks on different layers",
				input: [][]heartbeat.Heartbeater{
					{simpleHeartbeater{heartbeaterCheckResult{10, true, nil, false, "first error", []string{"tag"}}}},
					{simpleHeartbeater{heartbeaterCheckResult{15, true, nil, false, "didn't executed check", []string{}}}},
				},
				expected: graphExecutionResult{10, 0, true, false, []string{"first error"}, []string{"tag"}},
			},
		}

		for _, testCase := range testCases {
			Convey(fmt.Sprintf("%v", testCase.desc), func() {
				actual, err := testCase.input.executeGraph(0)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, testCase.expected)
			})
		}
	})
}

type simpleHeartbeater struct {
	checkResult heartbeaterCheckResult
}

func (h simpleHeartbeater) Check(int64) (int64, bool, error) {
	return h.checkResult.lastSuccessCheckElapsedTime, h.checkResult.hasErrors, h.checkResult.error
}

func (h simpleHeartbeater) NeedTurnOffNotifier() bool {
	return false
}

func (h simpleHeartbeater) NeedToCheckOthers() bool {
	return false
}

func (h simpleHeartbeater) GetErrorMessage() string {
	return h.checkResult.errorMessage
}

func (h simpleHeartbeater) GetCheckTags() heartbeat.CheckTags {
	return h.checkResult.checkTags
}
