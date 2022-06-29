package checker

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/metrics"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestCheckTemp(t *testing.T) {
	var logger, _ = logging.GetLogger("Checker_Test")
	dataBase := redis.NewTestDatabase(logger)
	dataBase.Flush()
	//defer dataBase.Flush()
	//var lastCheckTest = moira.CheckData{
	//	Score:       6000,
	//	State:       moira.StateOK,
	//	Timestamp:   1504509981,
	//	Maintenance: 1552723340,
	//	Metrics: map[string]moira.MetricState{
	//		"metric1": {
	//			EventTimestamp: 1504449789,
	//			State:          moira.StateNODATA,
	//			Suppressed:     false,
	//			Timestamp:      1504509380,
	//			Values:         map[string]float64{},
	//		},
	//		"metric2": {
	//			EventTimestamp: 1504449789,
	//			State:          moira.StateNODATA,
	//			Suppressed:     false,
	//			Timestamp:      1504509380,
	//			Values:         map[string]float64{},
	//		},
	//		"metric3": {
	//			EventTimestamp: 1504449789,
	//			State:          moira.StateNODATA,
	//			Suppressed:     false,
	//			Timestamp:      1504509380,
	//			Values:         map[string]float64{},
	//		},
	//		"metric4": {
	//			EventTimestamp: 1504463770,
	//			State:          moira.StateNODATA,
	//			Suppressed:     false,
	//			Timestamp:      1504509380,
	//			Values:         map[string]float64{},
	//		},
	//		"metric5": {
	//			EventTimestamp: 1504463770,
	//			State:          moira.StateNODATA,
	//			Suppressed:     false,
	//			Timestamp:      1504509380,
	//			Values:         map[string]float64{},
	//		},
	//		"metric6": {
	//			EventTimestamp: 1504463770,
	//			State:          "Ok",
	//			Suppressed:     false,
	//			Timestamp:      1504509380,
	//			Values:         map[string]float64{},
	//		},
	//		"metric7": {
	//			EventTimestamp: 1504463770,
	//			State:          "Ok",
	//			Suppressed:     false,
	//			Timestamp:      1504509380,
	//			Values:         map[string]float64{},
	//		},
	//	},
	//	MetricsToTargetRelation: map[string]string{},
	//}
	//dataBase.SetTriggerLastCheck(lastCheckTest)

	var errorValue float64 = 1
	var expression = ""
	const pattern1 = "Infrastructure.Production.Singular.*.*.kestrel.internalErrorsCount.*"
	const pattern2 = "Infrastructure.Production.Singular.*.*.request.total"
	var trigger = moira.Trigger{
		ID:             "e16ec022-7e1c-4559-815f-ac5926aef3f7",
		Name:           "TEST: Singular (prod): internal Kestel errors percent (for all clusters)",
		Targets:        []string{"alias(movingMin(scale(divideSeries(sumSeries(Infrastructure.Production.Singular.*.*.kestrel.internalErrorsCount.*), sumSeries(Infrastructure.Production.Singular.*.*.request.total)), 100), '3min'), \"Kestrel errors percent\")"},
		Tags:           []string{"Singular", "KE.Infra", "Infra.Comms.Critical"},
		Patterns:       []string{pattern1, pattern2},
		TriggerType:    moira.RisingTrigger,
		TTLState:       &moira.TTLStateNODATA,
		AloneMetrics:   map[string]bool{},
		ErrorValue:     &errorValue,
		TTL:            600,
		MuteNewMetrics: true,
		Expression:     &expression,
		Schedule: &moira.ScheduleData{
			Days: []moira.ScheduleDataDay{
				{
					Enabled: true,
					Name:    "mon",
				},
				{
					Enabled: true,
					Name:    "tue",
				},
				{
					Enabled: true,
					Name:    "wed",
				},
				{
					Enabled: true,
					Name:    "thu",
				},
				{
					Enabled: true,
					Name:    "fri",
				},
				{
					Enabled: true,
					Name:    "sat",
				},
				{
					Enabled: true,
					Name:    "sun",
				},
			},
			TimezoneOffset: -300,
			StartOffset:    0,
			EndOffset:      1439,
		},
	}
	const metric1 = "Infrastructure.Production.Singular.portal.dtl-por-snglr1.kestrel.internalErrorsCount.RequestLineTooLong"
	const metric2 = "Infrastructure.Production.Singular.k8s.bst-k8s-n02.request.total"

	matchedMetric1 := &moira.MatchedMetric{
		Patterns:           []string{pattern1},
		Metric:             metric1,
		Retention:          10,
		RetentionTimestamp: 10,
		Timestamp:          15,
		Value:              1,
	}
	matchedMetric2 := &moira.MatchedMetric{
		Patterns:           []string{pattern2},
		Metric:             metric2,
		Retention:          10,
		RetentionTimestamp: 20,
		Timestamp:          24,
		Value:              1,
	}

	var lastCheckTest = moira.CheckData{
		Score:                        1000,
		State:                        moira.StateOK,
		Timestamp:                    1656506370,
		LastSuccessfulCheckTimestamp: 1656506370,
		EventTimestamp:               1648609715,
		Metrics: map[string]moira.MetricState{
			"Kestrel errors percent": {
				EventTimestamp: 1656481110,
				State:          moira.StateNODATA,
				Suppressed:     false,
				Timestamp:      1656505948,
				Values:         map[string]float64{},
			},
		},
		MetricsToTargetRelation: map[string]string{},
	}

	Convey("Trigger manipulation", t, func() {
		err := dataBase.SaveTrigger(trigger.ID, &trigger)
		So(err, ShouldBeNil)

		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric1: matchedMetric1})
		So(err, ShouldBeNil)
		err = dataBase.SaveMetrics(map[string]*moira.MatchedMetric{metric2: matchedMetric2})
		So(err, ShouldBeNil)

		//actualValues, err := dataBase.GetMetricsValues([]string{metric1, metric2}, 0, time.Now().UTC().Unix())
		//fmt.Println(actualValues)
		//fmt.Println("11111111111")

		err = dataBase.SetTriggerLastCheck(trigger.ID, &lastCheckTest, false)
		So(err, ShouldBeNil)

		localSource := local.Create(dataBase)
		config := &checker.Config{}
		triggerChecker, err := checker.MakeTriggerChecker(trigger.ID, dataBase, logger, config, metricSource.CreateMetricSourceProvider(localSource, nil), &metrics.CheckerMetrics{})
		So(err, ShouldBeNil)
		_ = triggerChecker.Check()
	})
}
