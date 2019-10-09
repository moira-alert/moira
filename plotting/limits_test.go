package plotting

import (
	"math/rand"
	"testing"
	"time"

	"github.com/beevee/go-chart"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
	. "github.com/smartystreets/goconvey/convey"
)

// TestResolveLimits tests plot limits will be calculated correctly for any metricData array
func TestResolveLimits(t *testing.T) {
	var minValue = -1
	var maxValue = 10000
	stepTime := 60
	elementsToUse := 10
	startTime := time.Now().UTC().Unix()
	var metricsData []metricSource.MetricData
	// Fill MetricsData with random float64 values that higher than minValue and lower than maxValue
	for i := 0; i < int(elementsToUse); i++ {
		values := make([]float64, elementsToUse)
		for valInd := range values {
			values[valInd] = float64(rand.Intn(maxValue-1)) * rand.Float64()
		}
		metricData := *metricSource.MakeMetricData("test", values, int64(stepTime), int64(startTime))
		metricsData = append(metricsData, metricData)
	}
	// Change 2 first points of MetricsData to minValue and maxValue
	metricsData[0].Values[0], metricsData[0].Values[1] = float64(minValue), float64(maxValue)
	// So we're actually using elementsToUse x elementsToUse MetricsData with values like:
	// [-1                 10000              2252.779679004119  1333.7005695781143...
	// [1491.4815658695925 3528.452470599303  296.78548099273524 2048.473675536235
	// [1961.4869744066652 574.2827161848164  1757.8304749568863 2406.1508870508073
	// [2075.6933900571207 3393.385674988974  3234.9526818050126 5602.5371761246915
	// [384.016924791711   9066.931651012908  563.0027804705013  5100.243298996324
	// [519.4539685177408  6029.673742973381  243.3464382654782  2590.9614772639184
	// [3712.024207074801  8137.757246113409  3653.0361832312265 632.2809306369263
	// ...
	Convey("Resolve limits for collection of random MetricDatas", t, func() {
		expectedFrom := moira.Int64ToTime(int64(startTime))
		expectedTo := expectedFrom.Add(time.Duration(elementsToUse) * time.Minute)
		expectedIncrement := percentsOfRange(float64(minValue), float64(maxValue), defaultYAxisRangePercent)
		expectedLowest := float64(minValue) - expectedIncrement
		expectedHighest := float64(maxValue) + expectedIncrement
		limits := resolveLimits(metricsData)
		So(limits.from, ShouldResemble, expectedFrom)
		So(limits.to, ShouldResemble, expectedTo)
		So(limits.lowest, ShouldNotEqual, 0)
		So(limits.highest, ShouldNotEqual, 0)
		So(limits.lowest, ShouldNotEqual, limits.highest)
		So(limits.lowest, ShouldEqual, expectedLowest)
		So(limits.highest, ShouldEqual, expectedHighest)
	})
}

// TestGetThresholdAxisRange tests getThresholdAxisRange returns correct axis range
func TestGetThresholdAxisRange(t *testing.T) {
	testLimits := plotLimits{highest: 100, lowest: -100}
	Convey("Revert area between threshold line and x axis if necessary", t, func() {
		axisRange := testLimits.getThresholdAxisRange(moira.RisingTrigger)
		So(axisRange, ShouldResemble, chart.ContinuousRange{
			Descending: true,
			Max:        200,
			Min:        0,
		})
		nonRisingTriggers := []string{moira.FallingTrigger, moira.ExpressionTrigger}
		for _, triggerType := range nonRisingTriggers {
			axisRange = testLimits.getThresholdAxisRange(triggerType)
			So(axisRange, ShouldResemble, chart.ContinuousRange{
				Descending: false,
				Max:        100,
				Min:        -100,
			})
		}
	})
}

// TestFormsSetContaining tests formsSetContaining checks points correctly
func TestFormsSetContaining(t *testing.T) {
	Convey("check if point belongs to a given set", t, func() {
		testLimits := plotLimits{highest: 100, lowest: -100}
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
