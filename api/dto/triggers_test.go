package dto

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"regexp/syntax"
	"slices"
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

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, "/api/trigger", nil)
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
				ValidationError: fmt.Errorf("trigger name too long, should not be less than %d symbols", limit.Trigger.MaxNameSize),
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

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, "/api/trigger", nil)
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

			Convey("regexps in pattern", func() {
				type testcase struct {
					givenTargets   []string
					expectedErrRsp error
					caseDesc       string
				}

				testcases := []testcase{
					{
						givenTargets:   []string{"seriesByTag('name=some.metric', 'Team=Moira', 'Env=~Env1|Env2')"},
						expectedErrRsp: nil,
						caseDesc:       "with ' and at the end of query",
					},
					{
						givenTargets:   []string{"seriesByTag(\"name=some.metric\", \"Team=Moira\", \"Env=~Env1|Env2\")"},
						expectedErrRsp: nil,
						caseDesc:       "with \" and at the end of query",
					},
					{
						givenTargets:   []string{"seriesByTag('name=some.metric', 'Env=~Env1|Env2', 'Team=Moira')"},
						expectedErrRsp: nil,
						caseDesc:       "with ' in the middle of query",
					},
					{
						givenTargets:   []string{"seriesByTag(\"name=some.metric\", \"Env=~Env1|Env2\", \"Team=Moira\")"},
						expectedErrRsp: nil,
						caseDesc:       "with \" in the middle of query",
					},
					{
						givenTargets:   []string{"seriesByTag('name=some.metric', 'Env=~Env1|Env2'   , 'Team=Moira')"},
						expectedErrRsp: nil,
						caseDesc:       "in the middle of query with some spaces",
					},
					{
						givenTargets:   []string{"seriesByTag('name=some.metric', \"Vasya=~.*\" , 'Team=Moira', 'Env=~Env1|Env2')"},
						expectedErrRsp: nil,
						caseDesc:       "more than one regexp",
					},
					{
						givenTargets: []string{"seriesByTag('name=some.metric', \"Vasya=~+\", 'Team=Moira', 'BestTeam=Moira', 'Env=~Env1|Env2')"},
						expectedErrRsp: api.ErrInvalidRequestContent{
							ValidationError: fmt.Errorf(
								"bad regexp in tag 'Vasya': %w",
								&syntax.Error{
									Code: syntax.ErrMissingRepeatArgument,
									Expr: "+",
								}),
						},
						caseDesc: "with bad regexp (only '+')",
					},
					{
						givenTargets: []string{"seriesByTag('name=some.metric', \"Vasya=~*\", 'Team=Moira', 'BestTeam=Moira', 'Env=~Env1|Env2')"},
						expectedErrRsp: api.ErrInvalidRequestContent{
							ValidationError: fmt.Errorf(
								"bad regexp in tag 'Vasya': %w",
								&syntax.Error{
									Code: syntax.ErrMissingRepeatArgument,
									Expr: "*",
								}),
						},
						caseDesc: "with bad regexp (only '*')",
					},
					{
						givenTargets:   []string{"seriesByTag('name=some.metric', \"Vasya=~\" , 'Team=Moira', 'Env=~Env1|Env2')"},
						expectedErrRsp: nil,
						caseDesc:       "with empty regexp",
					},
					{
						givenTargets: []string{"seriesByTag('name=another.metric','Env=Env3','App=Moira','op=~*POST*')"},
						expectedErrRsp: api.ErrInvalidRequestContent{
							ValidationError: fmt.Errorf(
								"bad regexp in tag 'op': %w",
								&syntax.Error{
									Code: syntax.ErrMissingRepeatArgument,
									Expr: "*",
								}),
						},
						caseDesc: "with bad regexp (incorrect use of '*')",
					},
					{
						givenTargets: []string{"seriesByTag('name=other.metric','Env=Env1', 'App=Moira-API', 'ResCode=~^(?!200)')"},
						expectedErrRsp: api.ErrInvalidRequestContent{
							ValidationError: fmt.Errorf(
								"bad regexp in tag 'ResCode': %w",
								&syntax.Error{
									Code: syntax.ErrInvalidPerlOp,
									Expr: "(?!",
								}),
						},
						caseDesc: "with bad regexp '(?1'",
					},
					{
						givenTargets: []string{"seriesByTag('name=other.metric','Env=Env1', 'App=Moira-API', 'ResCode=~(4**)')"},
						expectedErrRsp: api.ErrInvalidRequestContent{
							ValidationError: fmt.Errorf(
								"bad regexp in tag 'ResCode': %w",
								&syntax.Error{
									Code: syntax.ErrInvalidRepeatOp,
									Expr: "**",
								}),
						},
						caseDesc: "with bad regexp '(4**)'",
					},
				}

				for i, singleCase := range testcases {
					Convey(fmt.Sprintf("Case %v: %s", i+1, singleCase.caseDesc), func() {
						trigger.Targets = singleCase.givenTargets
						tr := Trigger{trigger, throttling}

						fetchResult.EXPECT().GetPatterns().Return(singleCase.givenTargets, nil).AnyTimes()

						err := tr.Bind(request)
						So(err, ShouldResemble, singleCase.expectedErrRsp)
					})
				}
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

func Test_checkScheduleFilling(t *testing.T) {
	Convey("Testing checking schedule filling", t, func() {
		defaultSchedule := moira.NewDefaultScheduleData()

		Convey("With valid schedule", func() {
			givenSchedule := moira.NewDefaultScheduleData()

			givenSchedule.Days[len(givenSchedule.Days)-1].Enabled = false
			givenSchedule.TimezoneOffset += 1
			givenSchedule.StartOffset += 1
			givenSchedule.EndOffset += 1

			gotSchedule, err := checkScheduleFilling(givenSchedule)

			So(err, ShouldBeNil)
			So(gotSchedule, ShouldResemble, givenSchedule)
		})

		Convey("With not all days, missing days filled with false", func() {
			days := moira.GetFilledScheduleDataDays(true)

			givenSchedule := &moira.ScheduleData{
				Days:           days[:len(days)-1],
				TimezoneOffset: defaultSchedule.TimezoneOffset,
				StartOffset:    defaultSchedule.StartOffset,
				EndOffset:      defaultSchedule.EndOffset,
			}

			days[len(days)-1].Enabled = false

			expectedSchedule := &moira.ScheduleData{
				Days:           days,
				TimezoneOffset: defaultSchedule.TimezoneOffset,
				StartOffset:    defaultSchedule.StartOffset,
				EndOffset:      defaultSchedule.EndOffset,
			}

			gotSchedule, err := checkScheduleFilling(givenSchedule)

			So(err, ShouldBeNil)
			So(gotSchedule, ShouldResemble, expectedSchedule)
		})

		Convey("With some days repeated, there is no repeated days and missing days filled with false", func() {
			days := moira.GetFilledScheduleDataDays(true)

			days[4].Name = moira.Monday
			days[6].Name = moira.Monday

			givenSchedule := &moira.ScheduleData{
				Days:           days,
				TimezoneOffset: defaultSchedule.TimezoneOffset,
				StartOffset:    defaultSchedule.StartOffset,
				EndOffset:      defaultSchedule.EndOffset,
			}

			expectedDays := moira.GetFilledScheduleDataDays(true)

			expectedDays[4].Enabled = false
			expectedDays[6].Enabled = false

			expectedSchedule := &moira.ScheduleData{
				Days:           expectedDays,
				TimezoneOffset: defaultSchedule.TimezoneOffset,
				StartOffset:    defaultSchedule.StartOffset,
				EndOffset:      defaultSchedule.EndOffset,
			}

			gotSchedule, err := checkScheduleFilling(givenSchedule)

			So(err, ShouldBeNil)
			So(gotSchedule, ShouldResemble, expectedSchedule)
		})

		Convey("When days shuffled return ordered", func() {
			days := moira.GetFilledScheduleDataDays(true)

			shuffledDays := shuffleArray(days)

			givenSchedule := &moira.ScheduleData{
				Days:           shuffledDays,
				TimezoneOffset: defaultSchedule.TimezoneOffset,
				StartOffset:    defaultSchedule.StartOffset,
				EndOffset:      defaultSchedule.EndOffset,
			}

			expectedSchedule := &moira.ScheduleData{
				Days:           defaultSchedule.Days,
				TimezoneOffset: defaultSchedule.TimezoneOffset,
				StartOffset:    defaultSchedule.StartOffset,
				EndOffset:      defaultSchedule.EndOffset,
			}

			gotSchedule, err := checkScheduleFilling(givenSchedule)

			So(err, ShouldBeNil)
			So(gotSchedule, ShouldResemble, expectedSchedule)
		})

		Convey("When days shuffled and some are missed return ordered and filled missing", func() {
			days := moira.GetFilledScheduleDataDays(true)

			shuffledDays := shuffleArray(days[:len(days)-2])

			days[len(days)-1].Enabled = false
			days[len(days)-2].Enabled = false

			givenSchedule := &moira.ScheduleData{
				Days:           shuffledDays,
				TimezoneOffset: defaultSchedule.TimezoneOffset,
				StartOffset:    defaultSchedule.StartOffset,
				EndOffset:      defaultSchedule.EndOffset,
			}

			expectedSchedule := &moira.ScheduleData{
				Days:           days,
				TimezoneOffset: defaultSchedule.TimezoneOffset,
				StartOffset:    defaultSchedule.StartOffset,
				EndOffset:      defaultSchedule.EndOffset,
			}

			gotSchedule, err := checkScheduleFilling(givenSchedule)

			So(err, ShouldBeNil)
			So(gotSchedule, ShouldResemble, expectedSchedule)
		})

		Convey("With bad day names error returned", func() {
			days := moira.GetFilledScheduleDataDays(true)

			var (
				badMondayName moira.DayName = "Monday"
				badFridayName moira.DayName = "Friday"
			)

			days[0].Name = badMondayName
			days[4].Name = badFridayName

			givenSchedule := &moira.ScheduleData{
				Days:           days,
				TimezoneOffset: defaultSchedule.TimezoneOffset,
				StartOffset:    defaultSchedule.StartOffset,
				EndOffset:      defaultSchedule.EndOffset,
			}

			gotSchedule, err := checkScheduleFilling(givenSchedule)

			So(err, ShouldResemble, fmt.Errorf("bad day names in schedule: %s, %s", badMondayName, badFridayName))
			So(gotSchedule, ShouldBeNil)
		})

		Convey("With no enabled days error returned", func() {
			days := moira.GetFilledScheduleDataDays(false)

			givenSchedule := &moira.ScheduleData{
				Days:           days,
				TimezoneOffset: defaultSchedule.TimezoneOffset,
				StartOffset:    defaultSchedule.StartOffset,
				EndOffset:      defaultSchedule.EndOffset,
			}

			gotSchedule, err := checkScheduleFilling(givenSchedule)

			So(err, ShouldResemble, errNoAllowedDays)
			So(gotSchedule, ShouldBeNil)
		})
	})
}

func shuffleArray[S interface{ ~[]E }, E any](slice S) S {
	slice = slices.Clone(slice)
	shuffledSlice := make(S, 0, len(slice))

	for len(slice) > 0 {
		randomIdx := rand.Intn(len(slice))
		shuffledSlice = append(shuffledSlice, slice[randomIdx])

		switch {
		case randomIdx == len(slice)-1:
			slice = slice[:len(slice)-1]
		case randomIdx == 0:
			if len(slice) > 1 {
				slice = slice[1:]
			} else {
				slice = nil
			}
		default:
			slice = append(slice[:randomIdx], slice[randomIdx+1:]...)
		}
	}

	return shuffledSlice
}
