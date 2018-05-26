package plotting

import (
	"math"
	"testing"

	"github.com/go-graphite/carbonapi/expr/types"
	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
	. "github.com/smartystreets/goconvey/convey"
	"time"
)

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
	Convey("Resolve limits for given MetricsData for 5 minutes", t, func() {
		toTimeStamp := int32(1527330278)
		fromTimeStamp := int32(1527330278 - len(metricsData)*60)
		expectedTo := time.Date(2018, 5, 26, 10, 24, 38, 0, time.UTC)
		expectedFrom := expectedTo.Add(time.Duration(-len(metricsData)) * time.Minute)
		limits := ResolveLimits(metricsData, fromTimeStamp, toTimeStamp)
		So(limits.From, ShouldResemble, expectedFrom)
		So(limits.To, ShouldResemble, expectedTo)
		So(limits.Lowest, ShouldNotEqual, 0)
		So(limits.Highest, ShouldNotEqual, 0)
		So(limits.Lowest, ShouldNotEqual, limits.Highest)
		So(limits.Lowest, ShouldEqual, 1)
		So(limits.Highest, ShouldEqual, 10000)
	})
}

// TestFormsSetContaining tests FormsSetContaining checks points correctly
func TestFormsSetContaining(t *testing.T) {
	limits := Limits{
		Lowest:  -100,
		Highest: 100,
	}
	Convey("check if point belongs to a given set", t, func() {
		points := []float64{0, 10, 50, 100, 500, 1000}
		expectedResults := []bool{true, true, true, true, false, false}
		actualResults := make([]bool, 0)
		for _, point := range points {
			actualResult := limits.FormsSetContaining(point)
			actualResults = append(actualResults, actualResult)
		}
		So(len(actualResults), ShouldResemble, len(expectedResults))
		So(actualResults, ShouldResemble, expectedResults)
	})
}
