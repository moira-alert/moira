package plotting

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-graphite/carbonapi/expr/types"
	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
	"github.com/gotokatsuya/ipare"
	"github.com/gotokatsuya/ipare/util"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/wcharczuk/go-chart"
)

// TestGetRenderable renders plots based on test data and compares
// test plots hashes with plot examples hashes
func TestGetRenderable(t *testing.T) {
	Convey("Test DarkTheme", t, func() {
		Convey("Test simple plot", func() {
			Convey("No thresholds", func() {
				plotCase := "darkTheme.simple.noThresholds"
				generateAndSaveRenderable(plotCase, DarkTheme, nil, nil, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("Single Error threshold", func() {
				plotCase := "darkTheme.simple.single.errorThreshold"
				errorValue := float64(76)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, nil, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("Single Warn threshold", func() {
				plotCase := "darkTheme.simple.single.warnThreshold"
				warnValue := float64(57)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, &warnValue, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("isRaising threshold", func() {
				plotCase := "darkTheme.simple.isRaising.true"
				warnValue := float64(57)
				errorValue := float64(76)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("isFalling threshold", func() {
				plotCase := "darkTheme.simple.isRaising.false"
				warnValue := float64(76)
				errorValue := float64(57)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
		})
		Convey("Test humanized plot", func() {
			Convey("No thresholds", func() {
				plotCase := "darkTheme.humanized.noThresholds"
				generateAndSaveRenderable(plotCase, DarkTheme, nil, nil, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("Single Error threshold", func() {
				plotCase := "darkTheme.humanized.single.errorThreshold"
				errorValue := float64(76000)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, nil, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("Single Warn threshold", func() {
				plotCase := "darkTheme.humanized.single.warnThreshold"
				warnValue := float64(57000)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, &warnValue, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("isRaising threshold", func() {
				plotCase := "darkTheme.humanized.isRaising.true"
				warnValue := float64(57000)
				errorValue := float64(76000)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("isFalling threshold", func() {
				plotCase := "darkTheme.humanized.isRaising.false"
				warnValue := float64(76000)
				errorValue := float64(57000)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
		})
	})
	Convey("Test LightTheme", t, func() {
		Convey("Test simple plot", func() {
			Convey("No thresholds", func() {
				plotCase := "lightTheme.simple.noThresholds"
				generateAndSaveRenderable(plotCase, LightTheme, nil, nil, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("Single Error threshold", func() {
				plotCase := "lightTheme.simple.single.errorThreshold"
				errorValue := float64(76)
				generateAndSaveRenderable(plotCase, LightTheme, nil, nil, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("Single Warn threshold", func() {
				plotCase := "lightTheme.simple.single.warnThreshold"
				warnValue := float64(57)
				generateAndSaveRenderable(plotCase, LightTheme, nil, &warnValue, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("isRaising threshold", func() {
				plotCase := "lightTheme.simple.isRaising.true"
				warnValue := float64(57)
				errorValue := float64(76)
				generateAndSaveRenderable(plotCase, LightTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("isFalling threshold", func() {
				plotCase := "lightTheme.simple.isRaising.false"
				warnValue := float64(76)
				errorValue := float64(57)
				generateAndSaveRenderable(plotCase, LightTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
		})
		Convey("Test humanized plot", func() {
			Convey("No thresholds", func() {
				plotCase := "lightTheme.humanized.noThresholds"
				generateAndSaveRenderable(plotCase, LightTheme, nil, nil, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("Single Error threshold", func() {
				plotCase := "lightTheme.humanized.single.errorThreshold"
				errorValue := float64(76000)
				generateAndSaveRenderable(plotCase, LightTheme, nil, nil, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("Single Warn threshold", func() {
				plotCase := "lightTheme.humanized.single.warnThreshold"
				warnValue := float64(57000)
				generateAndSaveRenderable(plotCase, LightTheme, nil, &warnValue, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("isRaising threshold", func() {
				plotCase := "lightTheme.humanized.isRaising.true"
				warnValue := float64(57000)
				errorValue := float64(76000)
				generateAndSaveRenderable(plotCase, LightTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
			Convey("isFalling threshold", func() {
				plotCase := "lightTheme.humanized.isRaising.false"
				warnValue := float64(76000)
				errorValue := float64(57000)
				generateAndSaveRenderable(plotCase, LightTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldEqual, 0)
			})
		})
	})
}

func generateTestMetricsData(plotCase string) []*types.MetricData {
	metricData := types.MetricData{
		FetchResponse: pb.FetchResponse{
			Name:      "MetricName",
			StartTime: int32(0),
			StepTime:  int32(10),
			StopTime:  int32(100),
			Values:    []float64{12, 34, 23, 45, 76, 64, 32, 13, 34, 130, 70},
		},
	}
	metricData2 := types.MetricData{
		FetchResponse: pb.FetchResponse{
			Name:      "CategoryCounterType.MetricName",
			StartTime: int32(0),
			StepTime:  int32(10),
			StopTime:  int32(100),
			Values:    []float64{math.NaN(), 15, 32, math.NaN(), 54, 20, 43, 56, 2, 79, 76},
		},
	}
	metricData3 := types.MetricData{
		FetchResponse: pb.FetchResponse{
			Name:      "CategoryCounterName.CategoryCounterType.MetricName",
			StartTime: int32(0),
			StepTime:  int32(10),
			StopTime:  int32(100),
			Values:    []float64{11, 23, 45, math.NaN(), 45, math.NaN(), 32, 65, 78, 76, 74},
		},
	}
	metricData4 := types.MetricData{
		FetchResponse: pb.FetchResponse{
			Name:      "CategoryName.CategoryCounterName.CategoryCounterType.MetricName",
			StartTime: int32(0),
			StepTime:  int32(10),
			StopTime:  int32(100),
			Values:    []float64{11, 23, 10, 9, 17, 10, 25, 12, 10, 15, 30},
		},
	}
	if strings.Contains(plotCase, "humanized") {
		for valInd, value := range metricData.Values {
			metricData.Values[valInd] = 1000 * value
		}
		for valInd, value := range metricData2.Values {
			metricData2.Values[valInd] = 1000 * value
		}
		for valInd, value := range metricData3.Values {
			metricData3.Values[valInd] = 1000 * value
		}
		for valInd, value := range metricData4.Values {
			metricData4.Values[valInd] = 1000 * value
		}
	}
	metricsData := []*types.MetricData{&metricData, &metricData2, &metricData3, &metricData4}
	return metricsData
}

func generateAndSaveRenderable(plotCase string, plotTheme string, isRaising *bool, warnValue *float64, errorValue *float64) {
	metricsData := generateTestMetricsData(plotCase)
	font, _ := GetDefaultFont()
	plot := FromParams("triggerName", plotTheme, isRaising, warnValue, errorValue)
	renderable := plot.GetRenderable(metricsData, font)
	examples, _ := filepath.Abs("../plotting/examples/")
	fileName := fmt.Sprintf("%s/%s.test.png", examples, plotCase)
	f, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	w := bufio.NewWriter(f)
	if err := renderable.Render(chart.PNG, w); err != nil {
		panic(err)
	}
	w.Flush()
}

func calculateHashDistance(plotCase string) int {
	hash := ipare.NewHash()
	examples, _ := filepath.Abs("../plotting/examples/")
	pathToOriginal := fmt.Sprintf("%s/%s.png", examples, plotCase)
	pathToGenerated := fmt.Sprintf("%s/%s.test.png", examples, plotCase)
	original, _ := util.Open(pathToOriginal)
	generated, _ := util.Open(pathToGenerated)
	return hash.Compare(original, generated)
}
