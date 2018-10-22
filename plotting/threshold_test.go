package plotting

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// TestGenerateThresholds tests thresholds will be generated correctly
func TestGenerateThresholds(t *testing.T) {
	Convey("warn and error exists and belongs to set from limits", t, func() {
		plotLimits := Limits{
			From:    Int64ToTime(0),
			To:      Int64ToTime(100),
			Lowest:  0,
			Highest: 200,
		}
		Convey("warn < error", func() {
			warnValue := float64(100)
			errorValue := float64(200)
			plot := FromParams("", "", nil, &warnValue, &errorValue)
			plotThresholds := GenerateThresholds(plot, plotLimits, plot.IsRising())
			So(len(plotThresholds), ShouldEqual, 2)
			So(plotThresholds[0], ShouldResemble, Threshold{
				Title:     "ERROR",
				Value:     plotLimits.Highest - *plot.ErrorValue,
				TimePoint: float64(plotLimits.To.UnixNano()),
				Color:     ErrorThreshold,
				GrowTo:    0,
			})
			So(plotThresholds[1], ShouldResemble, Threshold{
				Title:     "WARN",
				Value:     plotLimits.Highest - *plot.WarnValue,
				TimePoint: float64(plotLimits.To.UnixNano()),
				Color:     WarningThreshold,
				GrowTo:    9,
			})
		})
		Convey("warn > error", func() {
			warnValue := float64(200)
			errorValue := float64(100)
			plot := FromParams("", "", nil, &warnValue, &errorValue)
			plotThresholds := GenerateThresholds(plot, plotLimits, plot.IsRising())
			So(len(plotThresholds), ShouldEqual, 2)
			So(plotThresholds[0], ShouldResemble, Threshold{
				Title:     "ERROR",
				Value:     *plot.ErrorValue,
				TimePoint: float64(plotLimits.To.UnixNano()),
				Color:     ErrorThreshold,
				GrowTo:    0,
			})
			So(plotThresholds[1], ShouldResemble, Threshold{
				Title:     "WARN",
				Value:     *plot.WarnValue,
				TimePoint: float64(plotLimits.To.UnixNano()),
				Color:     WarningThreshold,
				GrowTo:    9,
			})
		})
		Convey("warn = error", func() {
			warnValue := float64(100)
			errorValue := float64(100)
			plot := FromParams("", "", nil, &warnValue, &errorValue)
			plotThresholds := GenerateThresholds(plot, plotLimits, plot.IsRising())
			So(len(plotThresholds), ShouldEqual, 1)
			So(plotThresholds[0], ShouldResemble, Threshold{
				Title:     "ERROR",
				Value:     *plot.ErrorValue,
				TimePoint: float64(plotLimits.To.UnixNano()),
				Color:     ErrorThreshold,
				GrowTo:    0,
			})
		})
		Convey("warn ~< error", func() {
			warnValue := float64(100)
			errorValue := float64(110)
			plot := FromParams("", "", nil, &warnValue, &errorValue)
			plotThresholds := GenerateThresholds(plot, plotLimits, plot.IsRising())
			So(len(plotThresholds), ShouldEqual, 1)
			So(plotThresholds[0], ShouldResemble, Threshold{
				Title:     "ERROR",
				Value:     plotLimits.Highest - *plot.ErrorValue,
				TimePoint: float64(plotLimits.To.UnixNano()),
				Color:     ErrorThreshold,
				GrowTo:    0,
			})
		})
		Convey("warn >~ error", func() {
			warnValue := float64(110)
			errorValue := float64(100)
			plot := FromParams("", "", nil, &warnValue, &errorValue)
			plotThresholds := GenerateThresholds(plot, plotLimits, plot.IsRising())
			So(len(plotThresholds), ShouldEqual, 1)
			So(plotThresholds[0], ShouldResemble, Threshold{
				Title:     "ERROR",
				Value:     *plot.ErrorValue,
				TimePoint: float64(plotLimits.To.UnixNano()),
				Color:     ErrorThreshold,
				GrowTo:    0,
			})
		})
	})
	Convey("warn and error exists and not belongs to set from limits", t, func() {
		plotLimits := Limits{
			From:    Int64ToTime(0),
			To:      Int64ToTime(100),
			Lowest:  1000,
			Highest: 2000,
		}
		Convey("warn > error", func() {
			warnValue := float64(200)
			errorValue := float64(100)
			plot := FromParams("", "", nil, &warnValue, &errorValue)
			plotThresholds := GenerateThresholds(plot, plotLimits, plot.IsRising())
			So(len(plotThresholds), ShouldEqual, 0)
		})
		Convey("warn < error", func() {
			warnValue := float64(100)
			errorValue := float64(200)
			plot := FromParams("", "", nil, &warnValue, &errorValue)
			plotThresholds := GenerateThresholds(plot, plotLimits, plot.IsRising())
			So(len(plotThresholds), ShouldEqual, 0)
		})
		Convey("warn = error", func() {
			warnValue := float64(100)
			errorValue := float64(100)
			plot := FromParams("", "", nil, &warnValue, &errorValue)
			plotThresholds := GenerateThresholds(plot, plotLimits, plot.IsRising())
			So(len(plotThresholds), ShouldEqual, 0)
		})
		Convey("warn ~< error", func() {
			warnValue := float64(100)
			errorValue := float64(110)
			plot := FromParams("", "", nil, &warnValue, &errorValue)
			plotThresholds := GenerateThresholds(plot, plotLimits, plot.IsRising())
			So(len(plotThresholds), ShouldEqual, 0)
		})
		Convey("warn >~ error", func() {
			warnValue := float64(110)
			errorValue := float64(100)
			plot := FromParams("", "", nil, &warnValue, &errorValue)
			plotThresholds := GenerateThresholds(plot, plotLimits, plot.IsRising())
			So(len(plotThresholds), ShouldEqual, 0)
		})
	})
	Convey("error exists and belongs to set from limits", t, func() {
		plotLimits := Limits{
			From:    Int64ToTime(0),
			To:      Int64ToTime(100),
			Lowest:  0,
			Highest: 200,
		}
		errorValue := float64(100)
		plot := FromParams("", "", nil, nil, &errorValue)
		plotThresholds := GenerateThresholds(plot, plotLimits, plot.IsRising())
		So(len(plotThresholds), ShouldEqual, 1)
		So(plotThresholds[0], ShouldResemble, Threshold{
			Title:     "ERROR",
			Value:     *plot.ErrorValue,
			TimePoint: float64(plotLimits.To.UnixNano()),
			Color:     ErrorThreshold,
			GrowTo:    0,
		})
	})
	Convey("error exists and not belongs to set from limits", t, func() {
		plotLimits := Limits{
			From:    Int64ToTime(0),
			To:      Int64ToTime(100),
			Lowest:  1000,
			Highest: 2000,
		}
		errorValue := float64(100)
		plot := FromParams("", "", nil, nil, &errorValue)
		plotThresholds := GenerateThresholds(plot, plotLimits, plot.IsRising())
		So(len(plotThresholds), ShouldEqual, 0)
	})
	Convey("warn exists and belongs to set from limits", t, func() {
		plotLimits := Limits{
			From:    Int64ToTime(0),
			To:      Int64ToTime(100),
			Lowest:  0,
			Highest: 200,
		}
		warnValue := float64(100)
		plot := FromParams("", "", nil, &warnValue, nil)
		plotThresholds := GenerateThresholds(plot, plotLimits, plot.IsRising())
		So(len(plotThresholds), ShouldEqual, 1)
		So(plotThresholds[0], ShouldResemble, Threshold{
			Title:     "WARN",
			Value:     *plot.WarnValue,
			TimePoint: float64(plotLimits.To.UnixNano()),
			Color:     WarningThreshold,
			GrowTo:    9,
		})
	})
	Convey("warn exists and not belongs to set from limits", t, func() {
		plotLimits := Limits{
			From:    Int64ToTime(0),
			To:      Int64ToTime(100),
			Lowest:  1000,
			Highest: 2000,
		}
		warnValue := float64(100)
		plot := FromParams("", "", nil, &warnValue, nil)
		plotThresholds := GenerateThresholds(plot, plotLimits, plot.IsRising())
		So(len(plotThresholds), ShouldEqual, 0)
	})
}
