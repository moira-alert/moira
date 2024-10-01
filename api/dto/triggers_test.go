package dto

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/middleware"
	metricSource "github.com/moira-alert/moira/metric_source"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"

	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestTriggerValidation(t *testing.T) {
	Convey("Test trigger name and tags", t, func() {
		trigger := Trigger{
			TriggerModel: TriggerModel{},
		}

		limit := api.GetTestLimitsConfig()

		request, _ := http.NewRequest("PUT", "/api/trigger", nil)
		request.Header.Set("Content-Type", "application/json")
		request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "limits", limit))

		Convey("with empty targets", func() {
			err := trigger.Bind(request)

			So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: errTargetsRequired})
		})

		trigger.Targets = []string{"foo.bar"}

		Convey("with empty tag in tag list", func() {
			trigger.Tags = []string{""}

			err := trigger.Bind(request)

			So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: errTagsRequired})
		})

		trigger.Tags = append(trigger.Tags, "tag1")

		Convey("with empty Name", func() {
			err := trigger.Bind(request)

			So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: errTriggerNameRequired})
		})

		Convey("with too long Name", func() {
			trigger.Name = strings.Repeat("ё", limit.Trigger.MaxNameSize+1)

			err := trigger.Bind(request)

			So(err, ShouldResemble, api.ErrInvalidRequestContent{
				ValidationError: fmt.Errorf("trigger name too long, should not be greater than %d symbols", limit.Trigger.MaxNameSize),
			})
		})
	})

	Convey("Tests targets, values and expression validation", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
		remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
		fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
		sourceProvider := metricSource.CreateTestMetricSourceProvider(localSource, remoteSource, nil)

		request, _ := http.NewRequest("PUT", "/api/trigger", nil)
		request.Header.Set("Content-Type", "application/json")
		ctx := request.Context()
		ctx = context.WithValue(ctx, middleware.ContextKey("metricSourceProvider"), sourceProvider)
		ctx = context.WithValue(ctx, middleware.ContextKey("limits"), api.GetTestLimitsConfig())
		request = request.WithContext(ctx)

		desc := "Graphite ClickHouse"
		tags := []string{"Normal", "DevOps", "DevOpsGraphite-duty"}
		throttling := int64(0)
		warnValue := float64(10)
		errorValue := float64(5)

		trigger := TriggerModel{
			ID:             "GraphiteStoragesFreeSpace",
			Name:           "Graphite storage free space low",
			Desc:           &desc,
			Tags:           tags,
			TTLState:       &moira.TTLStateNODATA,
			TTL:            600,
			Schedule:       moira.NewDefaultScheduleData(),
			TriggerSource:  moira.GraphiteLocal,
			ClusterId:      moira.DefaultCluster,
			MuteNewMetrics: false,
		}

		Convey("Test FallingTrigger", func() {
			localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
			localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
			fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

			trigger.TriggerType = moira.FallingTrigger

			Convey("and one target", func() {
				trigger.Targets = []string{
					"aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
				}
				Convey("and expression", func() {
					trigger.Expression = "(t1 < 10 && t2 < 10) ? WARN:OK" //nolint
					tr := Trigger{trigger, throttling}
					err := tr.Bind(request)
					So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("can't use 'expression' to trigger_type: 'falling'")})
				})

				Convey("and warn_value and error_value", func() {
					trigger.WarnValue = &warnValue
					trigger.ErrorValue = &errorValue
					tr := Trigger{trigger, throttling}
					err := tr.Bind(request)
					So(err, ShouldBeNil)
				})
			})

			Convey("and one multiple targets", func() {
				trigger.Targets = []string{
					"aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
					"aliasByNode(DevOps.system.sd2-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
					"aliasByNode(DevOps.system.bst-graphite01.disk.root.gigabyte_percentfree, 2, 4)",
					"aliasByNode(DevOps.system.dtl-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
				}
				trigger.WarnValue = &warnValue
				trigger.ErrorValue = &errorValue
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("can't use trigger_type not 'falling' for with multiple targets")})
			})
		})
		Convey("Test RisingTrigger", func() {
			localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
			localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
			fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

			trigger.TriggerType = moira.RisingTrigger

			Convey("and one target", func() {
				trigger.Targets = []string{
					"aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
				}
				Convey("and expression", func() {
					trigger.Expression = "(t1 < 10 && t2 < 10) ? WARN:OK"
					tr := Trigger{trigger, throttling}
					err := tr.Bind(request)
					So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("can't use 'expression' to trigger_type: 'rising'")})
				})

				Convey("and warn_value and error_value", func() {
					trigger.WarnValue = &errorValue
					trigger.ErrorValue = &warnValue
					tr := Trigger{trigger, throttling}
					err := tr.Bind(request)
					So(err, ShouldBeNil)
				})
			})

			Convey("and one multiple targets", func() {
				trigger.Targets = []string{
					"aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
					"aliasByNode(DevOps.system.sd2-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
					"aliasByNode(DevOps.system.bst-graphite01.disk.root.gigabyte_percentfree, 2, 4)",
					"aliasByNode(DevOps.system.dtl-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
				}
				trigger.WarnValue = &errorValue
				trigger.ErrorValue = &warnValue
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("can't use trigger_type not 'rising' for with multiple targets")})
			})
		})
		Convey("Test ExpressionTrigger", func() {
			localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
			localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
			fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

			trigger.TriggerType = moira.ExpressionTrigger
			trigger.Expression = "(t1 < 10 && t2 < 10) ? WARN:OK"
			trigger.Targets = []string{
				"aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
				"aliasByNode(DevOps.system.sd2-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
				"aliasByNode(DevOps.system.bst-graphite01.disk.root.gigabyte_percentfree, 2, 4)",
				"aliasByNode(DevOps.system.dtl-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
			}
			Convey("and warn_value", func() {
				trigger.WarnValue = &warnValue
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldNotBeNil)
				So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("can't use 'warn_value' on trigger_type: 'expression'")})
			})

			Convey("and error_value", func() {
				trigger.ErrorValue = &errorValue
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("can't use 'error_value' on trigger_type: 'expression'")})
			})
			Convey("and expression", func() {
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldBeNil)
			})
		})

		Convey("Test alone metrics", func() {
			localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
			localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
			fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

			trigger.Targets = []string{"test target", "test target 2"}
			trigger.Expression = "OK"
			Convey("are empty", func() {
				trigger.AloneMetrics = map[string]bool{}
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldBeNil)
			})
			Convey("trigger with only one target", func() {
				trigger.Targets = []string{"test target"}
				trigger.AloneMetrics = map[string]bool{"t1": true}
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldBeNil)
				So(tr.AloneMetrics, ShouldResemble, map[string]bool{})
			})
			Convey("are valid", func() {
				trigger.AloneMetrics = map[string]bool{"t1": true}
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldBeNil)
			})
			Convey("have invalid metric name", func() {
				trigger.AloneMetrics = map[string]bool{"ttt": true}
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: errBadAloneMetricName})
			})
			Convey("have more than 1 metric name but only 1 need", func() {
				trigger.AloneMetrics = map[string]bool{"t1 t2": true}
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: errBadAloneMetricName})
			})
			Convey("have target higher than total amount of targets", func() {
				trigger.AloneMetrics = map[string]bool{"t3": true}
				tr := Trigger{trigger, throttling}
				err := tr.Bind(request)
				So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: errAloneMetricTargetIndexOutOfRange})
			})
		})

		Convey("Test patterns", func() {
			localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
			localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

			trigger.Expression = "OK"
			Convey("do not have asterisk", func() {
				trigger.Targets = []string{"sumSeries(some.test.series.*)"}
				tr := Trigger{trigger, throttling}
				fetchResult.EXPECT().GetPatterns().Return([]string{"some.test.series.*"}, nil).AnyTimes()
				err := tr.Bind(request)
				So(err, ShouldBeNil)
			})
			Convey("have asterisk", func() {
				trigger.Targets = []string{"sumSeries(*)"}
				tr := Trigger{trigger, throttling}
				fetchResult.EXPECT().GetPatterns().Return([]string{"*"}, nil).AnyTimes()
				err := tr.Bind(request)
				So(err, ShouldResemble, api.ErrInvalidRequestContent{ValidationError: errAsteriskPatternNotAllowed})
			})
		})
	})
}

func TestTriggerModel_ToMoiraTrigger(t *testing.T) {
	Convey("Test transforms TriggerModel to moira.Trigger", t, func() {
		expression := "t1 >0 ? OK : ERROR"
		warnValue := 1.0
		errorValue := 2.0
		desc := "description of trigger"
		now := time.Date(2022, time.June, 7, 10, 0, 0, 0, time.UTC).Unix()
		triggerModel := &TriggerModel{
			ID:          "trigger-id",
			Name:        "trigger-name",
			Desc:        &desc,
			Targets:     []string{"t1", "t2"},
			WarnValue:   &warnValue,
			ErrorValue:  &errorValue,
			TriggerType: moira.FallingTrigger,
			Tags:        []string{"tag-1", "tag-2"},
			TTLState:    &moira.TTLStateOK,
			TTL:         1,
			Schedule: &moira.ScheduleData{
				Days: []moira.ScheduleDataDay{
					{
						Enabled: true,
						Name:    "mon",
					},
				},
				TimezoneOffset: 1,
				StartOffset:    1,
				EndOffset:      1,
			},
			Expression:     expression,
			Patterns:       []string{"pattern-1", "pattern-2"},
			TriggerSource:  moira.GraphiteRemote,
			ClusterId:      moira.DefaultCluster,
			MuteNewMetrics: true,
			AloneMetrics: map[string]bool{
				"t1": true,
			},
			CreatedAt: getDateTime(&now),
			UpdatedAt: getDateTime(&now),
		}

		expTrigger := &moira.Trigger{
			ID:          "trigger-id",
			Name:        "trigger-name",
			Desc:        &desc,
			Targets:     []string{"t1", "t2"},
			WarnValue:   &warnValue,
			ErrorValue:  &errorValue,
			TriggerType: moira.FallingTrigger,
			Tags:        []string{"tag-1", "tag-2"},
			TTLState:    &moira.TTLStateOK,
			TTL:         1,
			Schedule: &moira.ScheduleData{
				Days: []moira.ScheduleDataDay{
					{
						Enabled: true,
						Name:    "mon",
					},
				},
				TimezoneOffset: 1,
				StartOffset:    1,
				EndOffset:      1,
			},
			Expression:     &expression,
			Patterns:       []string{"pattern-1", "pattern-2"},
			TriggerSource:  moira.GraphiteRemote,
			ClusterId:      moira.DefaultCluster,
			MuteNewMetrics: true,
			AloneMetrics: map[string]bool{
				"t1": true,
			},
		}

		So(triggerModel.ToMoiraTrigger(), ShouldResemble, expTrigger)
	})
}

func TestCreateTriggerModel(t *testing.T) {
	Convey("Test TriggerModel creation", t, func() {
		expression := "t1 >0 ? OK : ERROR"
		warnValue := 1.0
		errorValue := 2.0
		desc := "description of trigger"
		trigger := &moira.Trigger{
			ID:          "trigger-id",
			Name:        "trigger-name",
			Desc:        &desc,
			Targets:     []string{"t1", "t2"},
			WarnValue:   &warnValue,
			ErrorValue:  &errorValue,
			TriggerType: moira.FallingTrigger,
			Tags:        []string{"tag-1", "tag-2"},
			TTLState:    &moira.TTLStateOK,
			TTL:         1,
			Schedule: &moira.ScheduleData{
				Days: []moira.ScheduleDataDay{
					{
						Enabled: true,
						Name:    "mon",
					},
				},
				TimezoneOffset: 1,
				StartOffset:    1,
				EndOffset:      1,
			},
			Expression:     &expression,
			Patterns:       []string{"pattern-1", "pattern-2"},
			TriggerSource:  moira.GraphiteRemote,
			ClusterId:      moira.DefaultCluster,
			MuteNewMetrics: true,
			AloneMetrics: map[string]bool{
				"t1": true,
			},
		}

		expTriggerModel := TriggerModel{
			ID:          "trigger-id",
			Name:        "trigger-name",
			Desc:        &desc,
			Targets:     []string{"t1", "t2"},
			WarnValue:   &warnValue,
			ErrorValue:  &errorValue,
			TriggerType: moira.FallingTrigger,
			Tags:        []string{"tag-1", "tag-2"},
			TTLState:    &moira.TTLStateOK,
			TTL:         1,
			Schedule: &moira.ScheduleData{
				Days: []moira.ScheduleDataDay{
					{
						Enabled: true,
						Name:    "mon",
					},
				},
				TimezoneOffset: 1,
				StartOffset:    1,
				EndOffset:      1,
			},
			Expression:     expression,
			Patterns:       []string{"pattern-1", "pattern-2"},
			TriggerSource:  moira.GraphiteRemote,
			ClusterId:      moira.DefaultCluster,
			IsRemote:       true,
			MuteNewMetrics: true,
			AloneMetrics: map[string]bool{
				"t1": true,
			},
		}

		So(CreateTriggerModel(trigger), ShouldResemble, expTriggerModel)
	})
}
