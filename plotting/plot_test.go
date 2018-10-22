package plotting

import (
	"bufio"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-graphite/carbonapi/expr/types"
	pb "github.com/go-graphite/protocol/carbonapi_v3_pb"
	"github.com/gotokatsuya/ipare"
	"github.com/gotokatsuya/ipare/util"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/wcharczuk/go-chart"
)

// BenchmarkGetRenderableFromApi is a simple api rendering benchmark
func BenchmarkGetRenderableFromApi(b *testing.B) {
	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				warnValue := float64(57000)
				errorValue := float64(76000)
				metricsData := generateTestMetricsData("humanized")
				font, _ := GetDefaultFont()
				plot := FromParams("triggerName", DarkTheme, nil, &warnValue, &errorValue)
				renderable := plot.GetRenderable(metricsData, font)
				w.Header().Set("Content-Type", "image/png")
				renderable.Render(chart.PNG, w)
			},
		),
	)
	defer ts.Close()
	for i := 0; i < b.N; i++ {
		_, err := http.Get(ts.URL)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestGetRenderable renders plots based on test data and compares
// test plots hashes with plot examples hashes
func TestGetRenderable(t *testing.T) {
	Convey("Test DarkTheme", t, func() {
		Convey("Test simple plot", func() {
			Convey("No thresholds", func() {
				plotCase := "darkTheme.simple.noThresholds"
				generateAndSaveRenderable(plotCase, DarkTheme, nil, nil, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("Single Error threshold", func() {
				plotCase := "darkTheme.simple.single.errorThreshold"
				errorValue := float64(76)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, nil, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("Single Warn threshold", func() {
				plotCase := "darkTheme.simple.single.warnThreshold"
				warnValue := float64(57)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, &warnValue, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("isRising threshold", func() {
				plotCase := "darkTheme.simple.isRising.true"
				warnValue := float64(57)
				errorValue := float64(76)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("isFalling threshold", func() {
				plotCase := "darkTheme.simple.isRising.false"
				warnValue := float64(76)
				errorValue := float64(57)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
		})
		Convey("Test humanized plot", func() {
			Convey("No thresholds", func() {
				plotCase := "darkTheme.humanized.noThresholds"
				generateAndSaveRenderable(plotCase, DarkTheme, nil, nil, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("Single Error threshold", func() {
				plotCase := "darkTheme.humanized.single.errorThreshold"
				errorValue := float64(76000)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, nil, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("Single Warn threshold", func() {
				plotCase := "darkTheme.humanized.single.warnThreshold"
				warnValue := float64(57000)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, &warnValue, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("isRising threshold", func() {
				plotCase := "darkTheme.humanized.isRising.true"
				warnValue := float64(57000)
				errorValue := float64(76000)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("isFalling threshold", func() {
				plotCase := "darkTheme.humanized.isRising.false"
				warnValue := float64(76000)
				errorValue := float64(57000)
				generateAndSaveRenderable(plotCase, DarkTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
		})
	})
	Convey("Test LightTheme", t, func() {
		Convey("Test simple plot", func() {
			Convey("No thresholds", func() {
				plotCase := "lightTheme.simple.noThresholds"
				generateAndSaveRenderable(plotCase, LightTheme, nil, nil, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("Single Error threshold", func() {
				plotCase := "lightTheme.simple.single.errorThreshold"
				errorValue := float64(76)
				generateAndSaveRenderable(plotCase, LightTheme, nil, nil, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("Single Warn threshold", func() {
				plotCase := "lightTheme.simple.single.warnThreshold"
				warnValue := float64(57)
				generateAndSaveRenderable(plotCase, LightTheme, nil, &warnValue, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("isRising threshold", func() {
				plotCase := "lightTheme.simple.isRising.true"
				warnValue := float64(57)
				errorValue := float64(76)
				generateAndSaveRenderable(plotCase, LightTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("isFalling threshold", func() {
				plotCase := "lightTheme.simple.isRising.false"
				warnValue := float64(76)
				errorValue := float64(57)
				generateAndSaveRenderable(plotCase, LightTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
		})
		Convey("Test humanized plot", func() {
			Convey("No thresholds", func() {
				plotCase := "lightTheme.humanized.noThresholds"
				generateAndSaveRenderable(plotCase, LightTheme, nil, nil, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("Single Error threshold", func() {
				plotCase := "lightTheme.humanized.single.errorThreshold"
				errorValue := float64(76000)
				generateAndSaveRenderable(plotCase, LightTheme, nil, nil, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("Single Warn threshold", func() {
				plotCase := "lightTheme.humanized.single.warnThreshold"
				warnValue := float64(57000)
				generateAndSaveRenderable(plotCase, LightTheme, nil, &warnValue, nil)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("isRising threshold", func() {
				plotCase := "lightTheme.humanized.isRising.true"
				warnValue := float64(57000)
				errorValue := float64(76000)
				generateAndSaveRenderable(plotCase, LightTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
			Convey("isFalling threshold", func() {
				plotCase := "lightTheme.humanized.isRising.false"
				warnValue := float64(76000)
				errorValue := float64(57000)
				generateAndSaveRenderable(plotCase, LightTheme, nil, &warnValue, &errorValue)
				distance := calculateHashDistance(plotCase)
				So(distance, ShouldBeLessThanOrEqualTo, 5)
			})
		})
	})
}

// generateTestMetricsData generates metricData array for tests
func generateTestMetricsData(plotCase string) []*types.MetricData {
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

// generateAndSaveRenderable generates and saves rendered plots into a files
func generateAndSaveRenderable(plotCase string, plotTheme string, isRising *bool, warnValue *float64, errorValue *float64) {
	metricsData := generateTestMetricsData(plotCase)
	font, _ := GetDefaultFont()
	plot := FromParams("triggerName", plotTheme, isRising, warnValue, errorValue)
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

// calculateHashDistance returns calculated hash distance of two given pictures
func calculateHashDistance(plotCase string) int {
	hash := ipare.NewHash()
	examples, _ := filepath.Abs("../plotting/examples/")
	pathToOriginal := fmt.Sprintf("%s/%s.png", examples, plotCase)
	pathToGenerated := fmt.Sprintf("%s/%s.test.png", examples, plotCase)
	original, _ := util.Open(pathToOriginal)
	generated, _ := util.Open(pathToGenerated)
	return hash.Compare(original, generated)
}
