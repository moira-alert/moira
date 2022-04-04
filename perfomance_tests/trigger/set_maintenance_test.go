package trigger

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/golang/mock/gomock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"

	"github.com/gofrs/uuid"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
)

func BenchmarkSetTriggerCheckMaintenance(b *testing.B) {
	mockCtrl := gomock.NewController(b)
	defer mockCtrl.Finish()
	logger := getMockedLogger(mockCtrl)
	dataBase := redis.NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	triggerID := uuid.Must(uuid.NewV4()).String()
	lastCheckTest := generateLastCheck()
	err := dataBase.SetTriggerLastCheck(triggerID, lastCheckTest, false)
	if err != nil {
		b.Errorf("Can not set trigger last check: %s", err)
	}

	runBenchmark(b, dataBase, triggerID)
}

func getMockedLogger(mockCtrl *gomock.Controller) *mock_moira_alert.MockLogger {
	logger := mock_moira_alert.NewMockLogger(mockCtrl)
	logger.EXPECT().Clone().Return(logger).AnyTimes()
	logger.EXPECT().String(gomock.Any(), gomock.Any()).Return(logger).AnyTimes()
	logger.EXPECT().Infof(gomock.Any(), gomock.Any()).Return().AnyTimes()
	logger.EXPECT().Info(gomock.Any()).Return().AnyTimes()
	return logger
}

func generateLastCheck() *moira.CheckData {
	const minMetricsLength = 8
	const maxMetricsLength = 10
	var lastCheckTest = moira.CheckData{
		Score:                   6000,
		State:                   moira.StateOK,
		Timestamp:               1504509981,
		Maintenance:             1552723340,
		Metrics:                 map[string]moira.MetricState{},
		MetricsToTargetRelation: map[string]string{},
	}
	metricsSize := rand.Intn(maxMetricsLength-minMetricsLength) + minMetricsLength
	for i := 0; i < metricsSize; i++ {
		lastCheckTest.Metrics[fmt.Sprintf("metric%d", len(lastCheckTest.Metrics)+1)] = moira.MetricState{
			EventTimestamp: 1504449789,
			State:          moira.StateNODATA,
			Suppressed:     false,
			Timestamp:      1504509380,
			Values:         map[string]float64{"1": 1, "2": 2, "3": 3, "4": 4, "5": 5},
		}
	}
	return &lastCheckTest
}

func runBenchmark(b *testing.B, dataBase *redis.DbConnector, triggerID string) {
	var triggerMaintenanceTS int64 = 1000
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := dataBase.SetTriggerCheckMaintenance(triggerID, map[string]int64{"metric1": 1, "metric5": 5}, &triggerMaintenanceTS, "", 0)
		if err != nil {
			b.Errorf("Can not set trigger ckeck maintenance: %s", err)
		}
	}
}
