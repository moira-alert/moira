package plotting

import (
	"testing"

	"github.com/go-graphite/carbonapi/expr/types"
	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
	. "github.com/smartystreets/goconvey/convey"
)

// TestResolveLimits tests ResolveLimits on pseudo-random values
func TestResolveLimits(t *testing.T) {
	startTime := int32(0)
	stepTime := int32(15)
	stopTime := int32(180)
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
				StartTime: startTime,
				StepTime:  stepTime,
				StopTime:  stopTime,
			},
		}
		startTime ++
		stopTime ++
		metricsData = append(metricsData, &metricData)
	}
	Convey("Resolve limits for given MetricsData", t, func() {
		limits := ResolveLimits(metricsData)
		So(limits.From, ShouldResemble, Int32ToTime(0))
		So(limits.To, ShouldResemble, Int32ToTime(189))
		So(limits.Lowest, ShouldEqual, 11)
		So(limits.Highest, ShouldEqual, 450)
	})
}
