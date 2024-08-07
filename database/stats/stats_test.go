package stats

import (
	"testing"

	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestNewStatsManager(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	triggerStats := mock_moira_alert.NewMockStatsReporter(mockCtrl)
	contactStats := mock_moira_alert.NewMockStatsReporter(mockCtrl)

	Convey("Test new stats manager", t, func() {
		Convey("Successfully create new stats manager", func() {
			manager := NewStatsManager(triggerStats, contactStats)

			So(manager.reporters, ShouldResemble, []StatsReporter{triggerStats, contactStats})
		})

		Convey("Successfully start stats manager", func() {
			manager := NewStatsManager(triggerStats, contactStats)

			triggerStats.EXPECT().StartReport(manager.tomb.Dying()).Times(1)
			contactStats.EXPECT().StartReport(manager.tomb.Dying()).Times(1)

			manager.Start()
			defer manager.Stop() //nolint
		})
	})
}
