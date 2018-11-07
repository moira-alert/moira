package plotting

import (
	"math"
	"testing"
	"time"

	"github.com/go-graphite/carbonapi/expr/types"
	pb "github.com/go-graphite/protocol/carbonapi_v3_pb"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/wcharczuk/go-chart"

	"github.com/moira-alert/moira"
)

var testLimits = plotLimits{
	highest: 100,
	lowest:  -100,
}

// TestResolveLimits tests plot limits will be calculated correctly for any metricData array
func TestResolveLimits(t *testing.T) {
	metricsData := []*types.MetricData{
		{
			FetchResponse: pb.FetchResponse{
				Values: []float64{
					1, 2, 3, math.NaN(), 5,
				},
			},
		},
		{
			FetchResponse: pb.FetchResponse{
				Values: []float64{
					6, 7, math.NaN(), 9, 10,
				},
			},
		},
		{
			FetchResponse: pb.FetchResponse{
				Values: []float64{
					math.NaN(), 11, 12, 13, 10000,
				},
			},
		},
		{
			FetchResponse: pb.FetchResponse{
				Values: []float64{
					1, 1, 1, 1, 1,
				},
			},
		},
		{
			FetchResponse: pb.FetchResponse{
				Values: []float64{
					math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(),
				},
			},
		},
	}
	for _, metricData := range metricsData {
		metricData.StartTime = 1527329978
		metricData.StepTime = 60
		metricData.StopTime = 1527330278
	}
	Convey("Resolve limits for given MetricsData for 5 minutes", t, func() {
		expectedTo := time.Date(2018, 5, 26, 10, 24, 38, 0, time.UTC)
		expectedFrom := expectedTo.Add(time.Duration(-len(metricsData)) * time.Minute)
		limits := resolveLimits(metricsData)
		So(limits.from, ShouldResemble, expectedFrom)
		So(limits.to, ShouldResemble, expectedTo)
		So(limits.lowest, ShouldNotEqual, 0)
		So(limits.highest, ShouldNotEqual, 0)
		So(limits.lowest, ShouldNotEqual, limits.highest)
		So(limits.lowest, ShouldEqual, 1)
		So(limits.highest, ShouldEqual, 10000)
	})
}

// TestGetThresholdAxisRange tests getThresholdAxisRange returns correct axis range
func TestGetThresholdAxisRange(t *testing.T) {
	Convey("Revert area between threshold line and x axis if necessary", t, func() {
		axisRange := testLimits.getThresholdAxisRange(moira.RisingTrigger)
		So(axisRange, ShouldResemble, chart.ContinuousRange{
			Descending: true,
			Max: 200,
			Min: 0,
		})
		nonRisingTriggers := []string{moira.FallingTrigger, moira.ExpressionTrigger}
		for _, triggerType := range nonRisingTriggers {
			axisRange = testLimits.getThresholdAxisRange(triggerType)
			So(axisRange, ShouldResemble, chart.ContinuousRange{
				Descending: false,
				Max: 100,
				Min: -100,
			})
		}
	})
}

// TestFormsSetContaining tests formsSetContaining checks points correctly
func TestFormsSetContaining(t *testing.T) {
	Convey("check if point belongs to a given set", t, func() {
		points := []float64{0, 10, 50, 100, 101}
		expectedResults := []bool{true, true, true, true, false}
		actualResults := make([]bool, 0)
		for _, point := range points {
			actualResult := testLimits.formsSetContaining(point)
			actualResults = append(actualResults, actualResult)
		}
		So(len(actualResults), ShouldResemble, len(expectedResults))
		So(actualResults, ShouldResemble, expectedResults)
	})
}
