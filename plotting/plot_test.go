package plotting

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-graphite/carbonapi/expr/types"
	pb "github.com/go-graphite/protocol/carbonapi_v3_pb"
	"github.com/gotokatsuya/ipare"
	"github.com/gotokatsuya/ipare/util"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/wcharczuk/go-chart"

	"github.com/moira-alert/moira"
)

const (
	plottingExamplesPath        = "../plotting/_examples/"
	plottingExamplesPathPostfix = "example"
	maxAcceptableHashDistance   = 5
)

const (
	plotHashDistanceTestOuterPointIncrement   = float64(10000)
	plotHashDistanceTestOuterPointMultiplier  = float64(1000)
	plotHashDistanceTestRisingWarnThreshold   = float64(43)
	plotHashDistanceTestRisingErrorThreshold  = float64(76)
	plotHashDistanceTestFallingWarnThreshold  = plotHashDistanceTestRisingErrorThreshold
	plotHashDistanceTestFallingErrorThreshold = plotHashDistanceTestRisingWarnThreshold
)

// plotsHashDistancesTestCase is a single plot test case
type plotsHashDistancesTestCase struct {
	name               string
	plotTheme          string
	useHumanizedValues bool
	triggerType        string
	stateOk            bool
	warnValue          interface{}
	errorValue         interface{}
	expected           int
}

// getFilePath returns path to original or rendered plot file
func (testCase *plotsHashDistancesTestCase) getFilePath(toOriginal bool) (string, error) {
	examplesPath, err := filepath.Abs(plottingExamplesPath)
	if err != nil {
		return "", err
	}
	filePrefix := bytes.NewBuffer([]byte(examplesPath))
	filePrefix.WriteString(fmt.Sprintf("/%s.%s", testCase.plotTheme, testCase.triggerType))
	if testCase.stateOk {
		filePrefix.WriteString(".stateOk")
	} else {
		if testCase.warnValue != nil {
			filePrefix.WriteString(".warn")
		}
		if testCase.errorValue != nil {
			filePrefix.WriteString(".error")
		}
	}
	if !testCase.useHumanizedValues {
		filePrefix.WriteString(".humanized")
	}
	if toOriginal {
		return fmt.Sprintf("%s.%s.png", filePrefix.String(), plottingExamplesPathPostfix), nil
	}
	return fmt.Sprintf("%s.png", filePrefix.String()), nil
}

// getTriggerName returns test trigger name using plot test case parameters
func (testCase *plotsHashDistancesTestCase) getTriggerName() string {
	triggerName := bytes.NewBuffer([]byte("Test trigger ("))
	triggerName.WriteString(strings.ToUpper(string(testCase.plotTheme[0])))
	triggerName.WriteString(", ")
	triggerName.WriteString(strings.ToUpper(string(testCase.triggerType[0])))
	triggerName.WriteString(") {")
	if testCase.stateOk {
		triggerName.WriteString("OK")
	} else {
		if testCase.warnValue != nil {
			triggerName.WriteString("W+")
		} else {
			triggerName.WriteString("W-")
		}
		triggerName.WriteString(", ")
		if testCase.errorValue != nil {
			triggerName.WriteString("E+")
		} else {
			triggerName.WriteString("E-")
		}
	}
	triggerName.WriteString("}")
	if !testCase.useHumanizedValues {
		triggerName.WriteString(" [H]")
	}
	return triggerName.String()
}

// plotsHashDistancesTestCases is a collection of plot test cases
var plotsHashDistancesTestCases = []plotsHashDistancesTestCase{
	{
		name:               "DARK | EXPRESSION | No thresholds | Humanized values",
		plotTheme:          "dark",
		useHumanizedValues: true,
		triggerType:        moira.ExpressionTrigger,
		warnValue:          nil,
		errorValue:         nil,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | EXPRESSION | No thresholds | Not humanized values",
		plotTheme:          "dark",
		useHumanizedValues: false,
		triggerType:        moira.ExpressionTrigger,
		warnValue:          nil,
		errorValue:         nil,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | RISING | No thresholds | Humanized values",
		plotTheme:          "dark",
		useHumanizedValues: true,
		triggerType:        moira.RisingTrigger,
		stateOk:            true,
		warnValue:          plotHashDistanceTestRisingWarnThreshold - plotHashDistanceTestOuterPointIncrement,
		errorValue:         plotHashDistanceTestRisingErrorThreshold + plotHashDistanceTestOuterPointIncrement,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | RISING | WARN threshold | Humanized values",
		plotTheme:          "dark",
		useHumanizedValues: true,
		triggerType:        moira.RisingTrigger,
		warnValue:          plotHashDistanceTestRisingWarnThreshold,
		errorValue:         nil,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | RISING | ERROR threshold | Humanized values",
		plotTheme:          "dark",
		useHumanizedValues: true,
		triggerType:        moira.RisingTrigger,
		warnValue:          nil,
		errorValue:         plotHashDistanceTestRisingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | RISING | WARN and ERROR thresholds | Humanized values",
		plotTheme:          "dark",
		useHumanizedValues: true,
		triggerType:        moira.RisingTrigger,
		warnValue:          plotHashDistanceTestRisingWarnThreshold,
		errorValue:         plotHashDistanceTestRisingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | RISING | No thresholds | Not humanized values",
		plotTheme:          "dark",
		useHumanizedValues: false,
		triggerType:        moira.RisingTrigger,
		stateOk:            true,
		warnValue:          plotHashDistanceTestRisingWarnThreshold - plotHashDistanceTestOuterPointIncrement,
		errorValue:         plotHashDistanceTestRisingErrorThreshold + plotHashDistanceTestOuterPointIncrement,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | RISING | WARN threshold | Not humanized values",
		plotTheme:          "dark",
		useHumanizedValues: false,
		triggerType:        moira.RisingTrigger,
		warnValue:          plotHashDistanceTestRisingWarnThreshold,
		errorValue:         nil,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | RISING | ERROR threshold | Not humanized values",
		plotTheme:          "dark",
		useHumanizedValues: false,
		triggerType:        moira.RisingTrigger,
		warnValue:          nil,
		errorValue:         plotHashDistanceTestRisingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | RISING | WARN and ERROR thresholds | Not humanized values",
		plotTheme:          "dark",
		useHumanizedValues: false,
		triggerType:        moira.RisingTrigger,
		warnValue:          plotHashDistanceTestRisingWarnThreshold,
		errorValue:         plotHashDistanceTestRisingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | FALLING | No thresholds | Humanized values",
		plotTheme:          "dark",
		useHumanizedValues: true,
		triggerType:        moira.FallingTrigger,
		stateOk:            true,
		warnValue:          plotHashDistanceTestFallingWarnThreshold - plotHashDistanceTestOuterPointIncrement,
		errorValue:         plotHashDistanceTestFallingErrorThreshold + plotHashDistanceTestOuterPointIncrement,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | FALLING | WARN threshold | Humanized values",
		plotTheme:          "dark",
		useHumanizedValues: true,
		triggerType:        moira.FallingTrigger,
		warnValue:          plotHashDistanceTestFallingWarnThreshold,
		errorValue:         nil,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | FALLING | ERROR threshold | Humanized values",
		plotTheme:          "dark",
		useHumanizedValues: true,
		triggerType:        moira.FallingTrigger,
		warnValue:          nil,
		errorValue:         plotHashDistanceTestFallingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | FALLING | WARN and ERROR thresholds | Humanized values",
		plotTheme:          "dark",
		useHumanizedValues: true,
		triggerType:        moira.FallingTrigger,
		warnValue:          plotHashDistanceTestFallingWarnThreshold,
		errorValue:         plotHashDistanceTestFallingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | FALLING | No thresholds | Not humanized values",
		plotTheme:          "dark",
		useHumanizedValues: false,
		triggerType:        moira.FallingTrigger,
		stateOk:            true,
		warnValue:          plotHashDistanceTestFallingWarnThreshold - plotHashDistanceTestOuterPointIncrement,
		errorValue:         plotHashDistanceTestFallingErrorThreshold + plotHashDistanceTestOuterPointIncrement,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | FALLING | WARN threshold | Not humanized values",
		plotTheme:          "dark",
		useHumanizedValues: false,
		triggerType:        moira.FallingTrigger,
		warnValue:          plotHashDistanceTestFallingWarnThreshold,
		errorValue:         nil,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | FALLING | ERROR threshold | Not humanized values",
		plotTheme:          "dark",
		useHumanizedValues: false,
		triggerType:        moira.FallingTrigger,
		warnValue:          nil,
		errorValue:         plotHashDistanceTestFallingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "DARK | FALLING | WARN and ERROR thresholds | Not humanized values",
		plotTheme:          "dark",
		useHumanizedValues: false,
		triggerType:        moira.FallingTrigger,
		warnValue:          plotHashDistanceTestFallingWarnThreshold,
		errorValue:         plotHashDistanceTestFallingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | EXPRESSION | No thresholds | Humanized values",
		plotTheme:          "light",
		useHumanizedValues: true,
		triggerType:        moira.ExpressionTrigger,
		warnValue:          nil,
		errorValue:         nil,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | EXPRESSION | No thresholds | Not humanized values",
		plotTheme:          "light",
		useHumanizedValues: false,
		triggerType:        moira.ExpressionTrigger,
		warnValue:          nil,
		errorValue:         nil,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | RISING | No thresholds | Humanized values",
		plotTheme:          "light",
		useHumanizedValues: true,
		triggerType:        moira.RisingTrigger,
		stateOk:            true,
		warnValue:          plotHashDistanceTestRisingWarnThreshold - plotHashDistanceTestOuterPointIncrement,
		errorValue:         plotHashDistanceTestRisingErrorThreshold + plotHashDistanceTestOuterPointIncrement,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | RISING | WARN threshold | Humanized values",
		plotTheme:          "light",
		useHumanizedValues: true,
		triggerType:        moira.RisingTrigger,
		warnValue:          plotHashDistanceTestRisingWarnThreshold,
		errorValue:         nil,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | RISING | ERROR threshold | Humanized values",
		plotTheme:          "light",
		useHumanizedValues: true,
		triggerType:        moira.RisingTrigger,
		warnValue:          nil,
		errorValue:         plotHashDistanceTestRisingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | RISING | WARN and ERROR thresholds | Humanized values",
		plotTheme:          "light",
		useHumanizedValues: true,
		triggerType:        moira.RisingTrigger,
		warnValue:          plotHashDistanceTestRisingWarnThreshold,
		errorValue:         plotHashDistanceTestRisingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | RISING | No thresholds | Not humanized values",
		plotTheme:          "light",
		useHumanizedValues: false,
		triggerType:        moira.RisingTrigger,
		stateOk:            true,
		warnValue:          plotHashDistanceTestRisingWarnThreshold - plotHashDistanceTestOuterPointIncrement,
		errorValue:         plotHashDistanceTestRisingErrorThreshold + plotHashDistanceTestOuterPointIncrement,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | RISING | WARN threshold | Not humanized values",
		plotTheme:          "light",
		useHumanizedValues: false,
		triggerType:        moira.RisingTrigger,
		warnValue:          plotHashDistanceTestRisingWarnThreshold,
		errorValue:         nil,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | RISING | ERROR threshold | Not humanized values",
		plotTheme:          "light",
		useHumanizedValues: false,
		triggerType:        moira.RisingTrigger,
		warnValue:          nil,
		errorValue:         plotHashDistanceTestRisingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | RISING | WARN and ERROR thresholds | Not humanized values",
		plotTheme:          "light",
		useHumanizedValues: false,
		triggerType:        moira.RisingTrigger,
		warnValue:          plotHashDistanceTestRisingWarnThreshold,
		errorValue:         plotHashDistanceTestRisingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | FALLING | No thresholds | Humanized values",
		plotTheme:          "light",
		useHumanizedValues: true,
		triggerType:        moira.FallingTrigger,
		stateOk:            true,
		warnValue:          plotHashDistanceTestFallingWarnThreshold - plotHashDistanceTestOuterPointIncrement,
		errorValue:         plotHashDistanceTestFallingErrorThreshold + plotHashDistanceTestOuterPointIncrement,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | FALLING | WARN threshold | Humanized values",
		plotTheme:          "light",
		useHumanizedValues: true,
		triggerType:        moira.FallingTrigger,
		warnValue:          plotHashDistanceTestFallingWarnThreshold,
		errorValue:         nil,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | FALLING | ERROR threshold | Humanized values",
		plotTheme:          "light",
		useHumanizedValues: true,
		triggerType:        moira.FallingTrigger,
		warnValue:          nil,
		errorValue:         plotHashDistanceTestFallingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | FALLING | WARN and ERROR thresholds | Humanized values",
		plotTheme:          "light",
		useHumanizedValues: true,
		triggerType:        moira.FallingTrigger,
		warnValue:          plotHashDistanceTestFallingWarnThreshold,
		errorValue:         plotHashDistanceTestFallingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | FALLING | No thresholds | Not humanized values",
		plotTheme:          "light",
		useHumanizedValues: false,
		triggerType:        moira.FallingTrigger,
		stateOk:            true,
		warnValue:          plotHashDistanceTestFallingWarnThreshold - plotHashDistanceTestOuterPointIncrement,
		errorValue:         plotHashDistanceTestFallingErrorThreshold + plotHashDistanceTestOuterPointIncrement,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | FALLING | WARN threshold | Not humanized values",
		plotTheme:          "light",
		useHumanizedValues: false,
		triggerType:        moira.FallingTrigger,
		warnValue:          plotHashDistanceTestFallingWarnThreshold,
		errorValue:         nil,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | FALLING | ERROR threshold | Not humanized values",
		plotTheme:          "light",
		useHumanizedValues: false,
		triggerType:        moira.FallingTrigger,
		warnValue:          nil,
		errorValue:         plotHashDistanceTestFallingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
	{
		name:               "LIGHT | FALLING | WARN and ERROR thresholds | Not humanized values",
		plotTheme:          "light",
		useHumanizedValues: false,
		triggerType:        moira.FallingTrigger,
		warnValue:          plotHashDistanceTestFallingWarnThreshold,
		errorValue:         plotHashDistanceTestFallingErrorThreshold,
		expected:           maxAcceptableHashDistance,
	},
}

// generateTestMetricsData generates metricData array for tests
func generateTestMetricsData(useHumanizedValues bool) []*types.MetricData {
	metricData := types.MetricData{
		FetchResponse: pb.FetchResponse{
			Name:      "MetricName",
			StartTime: 0,
			StepTime:  10,
			StopTime:  100,
			Values:    []float64{12, 34, 23, 45, 76, 64, 32, 13, 34, 130, 70},
		},
	}
	metricData2 := types.MetricData{
		FetchResponse: pb.FetchResponse{
			Name:      "CategoryCounterType.MetricName",
			StartTime: 0,
			StepTime:  10,
			StopTime:  100,
			Values:    []float64{math.NaN(), 15, 32, math.NaN(), 54, 20, 43, 56, 2, 79, 76},
		},
	}
	metricData3 := types.MetricData{
		FetchResponse: pb.FetchResponse{
			Name:      "CategoryCounterName.CategoryCounterType.MetricName",
			StartTime: 0,
			StepTime:  10,
			StopTime:  100,
			Values:    []float64{11, 23, 45, math.NaN(), 45, math.NaN(), 32, 65, 78, 76, 74},
		},
	}
	metricData4 := types.MetricData{
		FetchResponse: pb.FetchResponse{
			Name:      "CategoryName.CategoryCounterName.CategoryCounterType.MetricName",
			StartTime: 0,
			StepTime:  10,
			StopTime:  100,
			Values:    []float64{11, 23, 10, 9, 17, 10, 25, 12, 10, 15, 30},
		},
	}
	if !useHumanizedValues {
		for valInd, value := range metricData.Values {
			metricData.Values[valInd] = plotHashDistanceTestOuterPointMultiplier * value
		}
		for valInd, value := range metricData2.Values {
			metricData2.Values[valInd] = plotHashDistanceTestOuterPointMultiplier * value
		}
		for valInd, value := range metricData3.Values {
			metricData3.Values[valInd] = plotHashDistanceTestOuterPointMultiplier * value
		}
		for valInd, value := range metricData4.Values {
			metricData4.Values[valInd] = plotHashDistanceTestOuterPointMultiplier * value
		}
	}
	metricsData := []*types.MetricData{&metricData, &metricData2, &metricData3, &metricData4}
	return metricsData
}

// renderTestMetricsDataToPNG renders and saves rendered plots to PNG
func renderTestMetricsDataToPNG(trigger moira.Trigger, plotTheme string,
	metricsData []*types.MetricData, filePath string) error {
	var metricsWhiteList []string
	location, _ := time.LoadLocation("UTC")
	plotTemplate, err := GetPlotTemplate(plotTheme, location)
	if err != nil {
		return err
	}
	renderable, err := plotTemplate.GetRenderable(&trigger, metricsData, metricsWhiteList)
	if err != nil {
		return err
	}
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	if err := renderable.Render(chart.PNG, w); err != nil {
		return err
	}
	w.Flush()
	return nil
}

// calculateHashDistance returns calculated hash distance of two given pictures
func calculateHashDistance(pathToOriginal, pathToRendered string) (*int, error) {
	hash := ipare.NewHash()
	original, err := util.Open(pathToOriginal)
	if err != nil {
		return nil, err
	}
	rendered, err := util.Open(pathToRendered)
	if err != nil {
		return nil, err
	}
	distance := hash.Compare(original, rendered)
	return &distance, nil
}

// TestGetRenderable renders plots based on test data and compares test plots hashes with plot examples hashes
func TestGetRenderable(t *testing.T) {
	Convey("Test plots hash distances", t, func() {
		for _, testCase := range plotsHashDistancesTestCases {
			trigger := moira.Trigger{
				Name: testCase.getTriggerName(),
				TriggerType: testCase.triggerType,
			}
			if testCase.errorValue != nil {
				errorValue := testCase.errorValue.(float64)
				if !testCase.useHumanizedValues {
					errorValue = errorValue * plotHashDistanceTestOuterPointMultiplier
				}
				trigger.ErrorValue = &errorValue
			}
			if testCase.warnValue != nil {
				warnValue := testCase.warnValue.(float64)
				if !testCase.useHumanizedValues {
					warnValue = warnValue * plotHashDistanceTestOuterPointMultiplier
				}
				trigger.WarnValue = &warnValue
			}
			metricsData := generateTestMetricsData(testCase.useHumanizedValues)
			pathToOriginal, err := testCase.getFilePath(true)
			if err != nil {
				t.Fatal(err)
			}
			pathToRendered, err := testCase.getFilePath(false)
			if err != nil {
				t.Fatal(err)
			}
			err = renderTestMetricsDataToPNG(trigger, testCase.plotTheme, metricsData, pathToRendered)
			if err != nil {
				t.Fatal(err)
			}
			hashDistance, err := calculateHashDistance(pathToOriginal, pathToRendered)
			if err != nil {
				t.Fatal(err)
			}
			os.Remove(pathToRendered)
			So(*hashDistance, ShouldBeLessThanOrEqualTo, testCase.expected)
		}
	})
}
