package moira

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestGetEventTags(testing *testing.T) {
	Convey("Progress should contains progress tag", testing, func() {
		event := NotificationEvent{
			State:    "OK",
			OldState: "WARN",
		}
		expected := []string{"OK", "WARN", "PROGRESS"}
		actual := event.GetEventTags()
		So(actual, ShouldResemble, expected)
	})

	Convey("Degradation should contains degradation tag", testing, func() {
		Convey("WARN -> OK", func() {
			event := NotificationEvent{
				State:    "WARN",
				OldState: "OK",
			}
			expected := []string{"WARN", "OK", "DEGRADATION"}
			actual := event.GetEventTags()
			So(actual, ShouldResemble, expected)
		})

		Convey("ERROR -> WARN", func() {
			event := NotificationEvent{
				State:    "ERROR",
				OldState: "WARN",
			}
			expected := []string{"ERROR", "WARN", "DEGRADATION"}
			actual := event.GetEventTags()
			So(actual, ShouldResemble, expected)
		})
	})

	Convey("High degradation should contains HIGH DEGRADATION tag", testing, func() {
		Convey("ERROR -> OK", func() {
			event := NotificationEvent{
				State:    "ERROR",
				OldState: "OK",
			}
			expected := []string{"ERROR", "OK", "HIGH DEGRADATION", "DEGRADATION"}
			actual := event.GetEventTags()
			So(actual, ShouldResemble, expected)
		})

		Convey("NODATA -> ERROR", func() {
			event := NotificationEvent{
				State:    "NODATA",
				OldState: "ERROR",
			}
			expected := []string{"NODATA", "ERROR", "HIGH DEGRADATION", "DEGRADATION"}
			actual := event.GetEventTags()
			So(actual, ShouldResemble, expected)
		})
	})

	Convey("Non-weighted test tag should contains test tag", testing, func() {
		event := NotificationEvent{
			State:    "TEST",
			OldState: "TEST",
		}
		expected := []string{"TEST", "TEST"}
		actual := event.GetEventTags()
		So(actual, ShouldResemble, expected)
	})
}
