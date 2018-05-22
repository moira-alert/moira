package plotting

import (
	"math/rand"
	"testing"

	"github.com/go-graphite/carbonapi/expr/types"
	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
	. "github.com/smartystreets/goconvey/convey"
)

// TestResolveLimits tests ResolveLimits on pseudo-random values
func TestResolveLimits(t *testing.T) {
	startTime := rand.Int31()
	stepTime := rand.Int31()
	stopTime := rand.Int31() * startTime * stepTime
	metricsData := make([]*types.MetricData, 0)
	metricsDataLength := 10
	for i := 0; i < metricsDataLength; i++ {
		metricData := types.MetricData{
			FetchResponse: pb.FetchResponse{
				Values: []float64{
					11, 23, 450, 47, 32,
				},
				IsAbsent: []bool{
					false, false, false, false, false,
				},
				StartTime: startTime + int32(i),
				StepTime:  stepTime + int32(i),
				StopTime:  stopTime + int32(i),
			},
		}
		metricsData = append(metricsData, &metricData)
	}
	Convey("Resolve limits for given MetricsData", t, func() {
		limits := ResolveLimits(metricsData)
		So(limits.From, ShouldResemble, Int32ToTime(startTime))
		So(limits.To, ShouldResemble, Int32ToTime(stopTime+int32(metricsDataLength-1)))
		So(limits.Lowest, ShouldEqual, 0)
		So(limits.Highest, ShouldEqual, 450)
	})
}
