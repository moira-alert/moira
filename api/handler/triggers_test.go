package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	dataBase "github.com/moira-alert/moira/database"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"

	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/xiam/to"
)

func TestGetSearchRequestString(t *testing.T) {
	Convey("Given a search request string", t, func() {
		Convey("The value should be converted into lower case", func() {
			testCases := []struct {
				text                  string
				expectedSearchRequest string
			}{
				{"query", "query"},
				{"QUERY", "query"},
				{"Query", "query"},
				{"QueRy", "query"},
			}
			for _, testCase := range testCases {
				req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("/api/trigger/search?onlyProblems=false&p=0&size=20&text=%s", testCase.text), nil)
				searchRequest := getSearchRequestString(req)
				So(searchRequest, ShouldEqual, testCase.expectedSearchRequest)
			}
		})
	})
}

func TestGetTriggerFromRequest(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
	sourceProvider := metricSource.CreateMetricSourceProvider(localSource, remoteSource, nil)

	localSource.EXPECT().IsConfigured().Return(true, nil).AnyTimes()
	localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
	localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
	fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
	fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

	Convey("Given a correct payload", t, func() {
		triggerWarnValue := 0.0
		triggerErrorValue := 1.0
		ttlState := moira.TTLState("NODATA")
		triggerDTO := dto.Trigger{
			TriggerModel: dto.TriggerModel{
				ID:             "test_id",
				Name:           "Test trigger",
				Desc:           new(string),
				Targets:        []string{"foo.bar"},
				WarnValue:      &triggerWarnValue,
				ErrorValue:     &triggerErrorValue,
				TriggerType:    "rising",
				Tags:           []string{"Normal", "DevOps", "DevOpsGraphite-duty"},
				TTLState:       &ttlState,
				TTL:            0,
				Schedule:       &moira.ScheduleData{},
				Expression:     "",
				Patterns:       []string{},
				TriggerSource:  moira.GraphiteLocal,
				MuteNewMetrics: false,
				AloneMetrics:   map[string]bool{},
				CreatedAt:      &time.Time{},
				UpdatedAt:      &time.Time{},
				CreatedBy:      "",
				UpdatedBy:      "anonymous",
			},
		}
		body, _ := json.Marshal(triggerDTO)

		request := httptest.NewRequest(http.MethodPut, "/trigger", bytes.NewReader(body))
		request.Header.Add("content-type", "application/json")
		request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))

		Convey("It should be parsed successfully", func() {
			trigger, err := getTriggerFromRequest(request)
			So(err, ShouldBeNil)
			So(trigger, ShouldResemble, &triggerDTO)
		})
	})

	Convey("Given an incorrect payload", t, func() {
		body := `{
			"name": "test",
			"desc": "",
			"targets": ["foo.bar"],
			"tags": ["test"],
			"patterns": [],
			"expression": "",
			"ttl": 600,
			"ttl_state": "NODATA",
			"sched": {
				"startOffset": 0,
				"endOffset": 1439,
				"tzOffset": -240,
				"days": null
			},
			"is_remote": false,
			"error_value": 1,
			"warn_value": 0,
			"trigger_type": "rising",
			"mute_new_metrics": false,
			"alone_metrics": "beliberda"
		}`

		request := httptest.NewRequest(http.MethodPut, "/trigger", strings.NewReader(body))
		request.Header.Add("content-type", "application/json")
		request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))

		Convey("Parser should return en error", func() {
			_, err := getTriggerFromRequest(request)
			So(err, ShouldHaveSameTypeAs, api.ErrorInvalidRequest(fmt.Errorf("")))
		})
	})
}

func TestGetMetricTTLByTrigger(t *testing.T) {
	request := httptest.NewRequest("", "/", strings.NewReader(""))
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "localMetricTTL", to.Duration("65m")))
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "remoteMetricTTL", to.Duration("168h")))

	Convey("Given a local trigger", t, func() {
		trigger := dto.Trigger{TriggerModel: dto.TriggerModel{
			TriggerSource: moira.GraphiteLocal,
		}}

		Convey("It's metric ttl should be equal to local", func() {
			So(getMetricTTLByTrigger(request, &trigger), ShouldEqual, 65*time.Minute)
		})
	})

	Convey("Given a remote trigger", t, func() {
		trigger := dto.Trigger{TriggerModel: dto.TriggerModel{
			TriggerSource: moira.GraphiteRemote,
		}}

		Convey("It's metric ttl should be equal to remote", func() {
			So(getMetricTTLByTrigger(request, &trigger), ShouldEqual, 168*time.Hour)
		})
	})
}

func TestTriggerCheckHandler(t *testing.T) {
	Convey("Test triggerCheck handler", t, func() {
		Convey("Checking target metric ttl validation", func() {
			mockCtrl := gomock.NewController(t)
			responseWriter := httptest.NewRecorder()

			localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
			remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
			fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
			sourceProvider := metricSource.CreateMetricSourceProvider(localSource, remoteSource, nil)

			localSource.EXPECT().IsConfigured().Return(true, nil).AnyTimes()
			localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
			localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
			fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

			remoteSource.EXPECT().IsConfigured().Return(true, nil).AnyTimes()
			remoteSource.EXPECT().GetMetricsTTLSeconds().Return(int64(604800)).AnyTimes()
			remoteSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()

			testCases := []struct {
				triggerSource    moira.TriggerSource
				targets          []string
				expectedResponse string
			}{
				{
					moira.GraphiteLocal,
					[]string{
						"integralByInterval(aliasSub(sum(aliasByNode(my.own.metric, 6)), '(.*)', 'metric'), '1h')",
					},
					"{\"targets\":[{\"syntax_ok\":true}]}\n",
				},
				{
					moira.GraphiteLocal,
					[]string{
						"integralByInterval(aliasSub(sum(aliasByNode(my.own.metric, 6)), '(.*)', 'metric'), '6h')",
					},
					"{\"targets\":[{\"syntax_ok\":true,\"tree_of_problems\":{\"argument\":\"integralByInterval\",\"position\":0,\"problems\":[{\"argument\":\"6h\",\"type\":\"bad\",\"description\":\"The function integralByInterval has a time sampling parameter 6h larger than allowed by the config:1h5m0s\",\"position\":1}]}}]}\n",
				},
				{
					moira.GraphiteLocal,
					[]string{
						"my.own.metric",
					},
					"{\"targets\":[{\"syntax_ok\":true}]}\n",
				},
				{
					moira.GraphiteRemote,
					[]string{
						"integralByInterval(aliasSub(sum(aliasByNode(my.own.metric, 6)), '(.*)', 'metric'), '1h')",
					},
					"{\"targets\":[{\"syntax_ok\":true}]}\n",
				},
				{
					moira.GraphiteRemote,
					[]string{
						"integralByInterval(aliasSub(sum(aliasByNode(my.own.metric, 6)), '(.*)', 'metric'), '6h')",
					},
					"{\"targets\":[{\"syntax_ok\":true}]}\n",
				},
			}
			for n, testCase := range testCases {
				Convey(fmt.Sprintf("TestCase #%d", n), func() {
					triggerWarnValue := float64(10)
					triggerErrorValue := float64(15)
					triggerDTO := dto.Trigger{
						TriggerModel: dto.TriggerModel{
							Name:          "Test trigger",
							Tags:          []string{"Normal", "DevOps", "DevOpsGraphite-duty"},
							WarnValue:     &triggerWarnValue,
							ErrorValue:    &triggerErrorValue,
							Targets:       testCase.targets,
							TriggerSource: testCase.triggerSource,
						},
					}
					jsonTrigger, _ := json.Marshal(triggerDTO)
					testRequest := httptest.NewRequest("", "/", bytes.NewBuffer(jsonTrigger))
					testRequest.Header.Add("content-type", "application/json")
					testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "metricSourceProvider", sourceProvider))
					testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "localMetricTTL", to.Duration("65m")))
					testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "remoteMetricTTL", to.Duration("168h")))

					triggerCheck(responseWriter, testRequest)

					response := responseWriter.Result()
					defer response.Body.Close()

					contentBytes, _ := io.ReadAll(response.Body)
					contents := string(contentBytes)

					So(contents, ShouldEqual, testCase.expectedResponse)
				})
			}
		})
	})
}

func TestCreateTriggerHandler(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	sourceProvider := metricSource.CreateMetricSourceProvider(localSource, remoteSource, nil)

	localSource.EXPECT().IsConfigured().Return(true, nil).AnyTimes()
	localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
	fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
	localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
	fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
	fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

	const validateFlag = "validate"

	mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)
	database = mockDb

	Convey("When createTrigger was called with normal input", t, func() {
		urls := []string{
			"/",
			fmt.Sprintf("/trigger?%s", validateFlag),
		}

		for _, url := range urls {
			mockDb.EXPECT().AcquireTriggerCheckLock(gomock.Any(), gomock.Any()).Return(nil)
			mockDb.EXPECT().DeleteTriggerCheckLock(gomock.Any())
			mockDb.EXPECT().GetTriggerLastCheck(gomock.Any())
			mockDb.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any())
			mockDb.EXPECT().SaveTrigger(gomock.Any(), gomock.Any())

			triggerWarnValue := float64(10)
			triggerErrorValue := float64(15)
			triggerDTO := dto.Trigger{
				TriggerModel: dto.TriggerModel{
					Name:          "Test trigger",
					Tags:          []string{"123"},
					WarnValue:     &triggerWarnValue,
					ErrorValue:    &triggerErrorValue,
					Targets:       []string{"my.metric"},
					TriggerSource: moira.GraphiteLocal,
				},
			}
			jsonTrigger, _ := json.Marshal(triggerDTO)
			testRequest := httptest.NewRequest("", url, bytes.NewBuffer(jsonTrigger))
			testRequest.Header.Add("content-type", "application/json")
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "metricSourceProvider", sourceProvider))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "localMetricTTL", to.Duration("65m")))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, testRequest)

			Convey(fmt.Sprintf("should return success message, url=%s", url), func() {
				response := responseWriter.Result()
				defer response.Body.Close()
				So(response.StatusCode, ShouldEqual, http.StatusOK)
				So(isTriggerCreated(response), ShouldBeTrue)
			})
		}
	})

	Convey("When createTrigger was called with empty targets", t, func() {
		urls := []string{
			"/",
			fmt.Sprintf("/trigger?%s", validateFlag),
		}

		for _, url := range urls {
			triggerWarnValue := float64(10)
			triggerErrorValue := float64(15)
			triggerDTO := dto.Trigger{
				TriggerModel: dto.TriggerModel{
					Name:          "Test trigger",
					Tags:          []string{"123"},
					WarnValue:     &triggerWarnValue,
					ErrorValue:    &triggerErrorValue,
					Targets:       []string{},
					TriggerSource: moira.GraphiteLocal,
				},
			}
			jsonTrigger, _ := json.Marshal(triggerDTO)
			request := httptest.NewRequest("", url, bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "localMetricTTL", to.Duration("65m")))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			Convey(fmt.Sprintf("should return 400, url=%s", url), func() {
				response := responseWriter.Result()
				defer response.Body.Close()
				So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
			})
		}
	})

	Convey("When createTrigger was called with target with warning function", t, func() {
		triggerWarnValue := float64(10)
		triggerErrorValue := float64(15)
		trigger := dto.Trigger{
			TriggerModel: dto.TriggerModel{
				Name:          "Test trigger",
				Tags:          []string{"123"},
				WarnValue:     &triggerWarnValue,
				ErrorValue:    &triggerErrorValue,
				Targets:       []string{"alias(consolidateBy(Sales.widgets.largeBlue, 'sum'), 'alias to test nesting')"},
				TriggerSource: moira.GraphiteLocal,
			},
		}
		jsonTrigger, _ := json.Marshal(trigger)

		Convey("without validate like before", func() {
			mockDb.EXPECT().AcquireTriggerCheckLock(gomock.Any(), gomock.Any()).Return(nil)
			mockDb.EXPECT().DeleteTriggerCheckLock(gomock.Any())
			mockDb.EXPECT().GetTriggerLastCheck(gomock.Any())
			mockDb.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any())
			mockDb.EXPECT().SaveTrigger(gomock.Any(), gomock.Any())

			request := httptest.NewRequest("", "/", bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "localMetricTTL", to.Duration("65m")))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			Convey("should return 200", func() {
				response := responseWriter.Result()
				defer response.Body.Close()
				So(response.StatusCode, ShouldEqual, http.StatusOK)
				So(isTriggerCreated(response), ShouldBeTrue)
			})
		})

		Convey("with validate", func() {
			mockDb.EXPECT().AcquireTriggerCheckLock(gomock.Any(), gomock.Any()).Return(nil)
			mockDb.EXPECT().DeleteTriggerCheckLock(gomock.Any())
			mockDb.EXPECT().GetTriggerLastCheck(gomock.Any())
			mockDb.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any())
			mockDb.EXPECT().SaveTrigger(gomock.Any(), gomock.Any())

			request := httptest.NewRequest("", fmt.Sprintf("/trigger?%s", validateFlag), bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "localMetricTTL", to.Duration("65m")))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			Convey("should return 200 and tree of problems", func() {
				response := responseWriter.Result()
				defer response.Body.Close()

				So(response.StatusCode, ShouldEqual, http.StatusOK)

				contentBytes, _ := io.ReadAll(response.Body)
				actual := dto.SaveTriggerResponse{}
				_ = json.Unmarshal(contentBytes, &actual)

				expectedTargets := []dto.TreeOfProblems{
					{
						SyntaxOk: true,
						TreeOfProblems: &dto.ProblemOfTarget{
							Argument: "alias",
							Problems: []dto.ProblemOfTarget{
								{
									Argument:    "consolidateBy",
									Type:        "warn",
									Description: "This function affects only visual graph representation. It is meaningless in Moira.",
								},
							},
						},
					},
				}
				So(actual.CheckResult.Targets, ShouldResemble, expectedTargets)
				const expected = "trigger created"
				So(actual.Message, ShouldEqual, expected)
			})
		})
	})

	Convey("When createTrigger was called with target with bad (error) function", t, func() {
		triggerWarnValue := float64(10)
		triggerErrorValue := float64(15)
		triggerDTO := dto.Trigger{
			TriggerModel: dto.TriggerModel{
				Name:          "Test trigger",
				Tags:          []string{"123"},
				WarnValue:     &triggerWarnValue,
				ErrorValue:    &triggerErrorValue,
				Targets:       []string{"alias(summarize(my.metric, '5min'), 'alias to test nesting')"},
				TriggerSource: moira.GraphiteLocal,
			},
		}
		jsonTrigger, _ := json.Marshal(triggerDTO)

		Convey("without validate like before", func() {
			mockDb.EXPECT().AcquireTriggerCheckLock(gomock.Any(), gomock.Any()).Return(nil)
			mockDb.EXPECT().DeleteTriggerCheckLock(gomock.Any())
			mockDb.EXPECT().GetTriggerLastCheck(gomock.Any())
			mockDb.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any())
			mockDb.EXPECT().SaveTrigger(gomock.Any(), gomock.Any())

			request := httptest.NewRequest("", "/", bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "localMetricTTL", to.Duration("65m")))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			Convey("should return 200", func() {
				response := responseWriter.Result()
				defer response.Body.Close()
				So(response.StatusCode, ShouldEqual, http.StatusOK)
				So(isTriggerCreated(response), ShouldBeTrue)
			})
		})

		Convey("with validate", func() {
			request := httptest.NewRequest("", fmt.Sprintf("/trigger?%s", validateFlag), bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "localMetricTTL", to.Duration("65m")))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			Convey("should return 400 and tree of problems", func() {
				response := responseWriter.Result()
				defer response.Body.Close()

				So(response.Header.Get("Content-Type"), ShouldEqual, "application/json; charset=utf-8")
				So(response.StatusCode, ShouldEqual, http.StatusBadRequest)

				contentBytes, _ := io.ReadAll(response.Body)
				actual := dto.SaveTriggerResponse{}
				_ = json.Unmarshal(contentBytes, &actual)

				expected := dto.SaveTriggerResponse{
					CheckResult: dto.TriggerCheckResponse{
						Targets: []dto.TreeOfProblems{
							{
								SyntaxOk: true,
								TreeOfProblems: &dto.ProblemOfTarget{
									Argument: "alias",
									Problems: []dto.ProblemOfTarget{
										{
											Argument:    "summarize",
											Type:        "bad",
											Description: "This function is unstable: it can return different historical values with each evaluation. Moira will show unexpected values that you don't see on your graphs.",
										},
									},
								},
							},
						},
					},
				}
				So(actual, ShouldResemble, expected)
			})
		})
	})
}

func TestTriggersCreatedWithTriggerSource(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	prometheusSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	sourceProvider := metricSource.CreateMetricSourceProvider(localSource, remoteSource, prometheusSource)

	db := mock_moira_alert.NewMockDatabase(mockCtrl)
	database = db
	defer func() { database = nil }()

	triggerId := "test"
	target := `test_target_value`

	Convey("Given is_remote flag is false and trigger_source is not set", t, func() {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"is_remote": false`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be graphite local", func() {
			setupExpectationsForCreateTrigger(localSource, db, target, triggerId, moira.GraphiteLocal)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	Convey("Given is_remote flag is true and trigger_source is not set", t, func() {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"is_remote": true`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be graphite remote", func() {
			setupExpectationsForCreateTrigger(remoteSource, db, target, triggerId, moira.GraphiteRemote)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	Convey("Given is_remote flag is not set and trigger_source is graphite_local", t, func() {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"trigger_source": "graphite_local"`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be graphite local", func() {
			setupExpectationsForCreateTrigger(localSource, db, target, triggerId, moira.GraphiteLocal)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	Convey("Given is_remote flag is not set and trigger_source is graphite_remote", t, func() {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"trigger_source": "graphite_remote"`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be graphite remote", func() {
			setupExpectationsForCreateTrigger(remoteSource, db, target, triggerId, moira.GraphiteRemote)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	Convey("Given is_remote flag is not set and trigger_source is prometheus_remote", t, func() {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"trigger_source": "prometheus_remote"`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be prometheus remote", func() {
			setupExpectationsForCreateTrigger(prometheusSource, db, target, triggerId, moira.PrometheusRemote)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	Convey("Given is_remote flag is true and trigger_source is graphite_local", t, func() {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"is_remote": true, "trigger_source": "graphite_local"`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be graphite local", func() {
			setupExpectationsForCreateTrigger(localSource, db, target, triggerId, moira.GraphiteLocal)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})
}

func makeTestTriggerJson(target, triggerId, triggerSource string) []byte {
	targetJson, _ := json.Marshal(target)
	jsonTrigger := fmt.Sprintf(`{
		"name": "Test",
		"targets": [ %s ],
		"id": "%s",
		"warn_value": 100,
		"error_value": 200,
		"trigger_type": "rising",
		"tags": [ "test" ],
		"ttl_state": "NODATA",
		%s,
		"ttl": 600
	}`, targetJson, triggerId, triggerSource)
	return []byte(jsonTrigger)
}

func setupExpectationsForCreateTrigger(
	source *mock_metric_source.MockMetricSource,
	db *mock_moira_alert.MockDatabase,
	target, triggerId string,
	triggerSource moira.TriggerSource,
) {
	source.EXPECT().IsConfigured().Return(true, nil)
	source.EXPECT().GetMetricsTTLSeconds().Return(int64(3600))
	source.EXPECT().Fetch(target, gomock.Any(), gomock.Any(), gomock.Any()).Return(&local.FetchResult{}, nil)

	db.EXPECT().GetTrigger(triggerId).Return(moira.Trigger{}, dataBase.ErrNil)
	db.EXPECT().AcquireTriggerCheckLock(triggerId, gomock.Any()).Return(nil)
	db.EXPECT().DeleteTriggerCheckLock(triggerId).Return(nil)
	db.EXPECT().GetTriggerLastCheck(triggerId).Return(moira.CheckData{}, dataBase.ErrNil)
	db.EXPECT().SetTriggerLastCheck(triggerId, gomock.Any(), triggerSource).Return(nil)
	db.EXPECT().SaveTrigger(triggerId, gomock.Any()).Return(nil)
}

func newTriggerCreateRequest(
	sourceProvider *metricSource.SourceProvider,
	triggerId string,
	jsonTrigger []byte,
) *http.Request {
	request := httptest.NewRequest("PUT", "/trigger", bytes.NewBuffer(jsonTrigger))
	request.Header.Add("content-type", "application/json")
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "localMetricTTL", to.Duration("65m")))
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), triggerIDKey, triggerId))

	return request
}

func isTriggerCreated(response *http.Response) bool {
	contentBytes, _ := io.ReadAll(response.Body)
	actual := dto.SaveTriggerResponse{}
	_ = json.Unmarshal(contentBytes, &actual)
	const expected = "trigger created"

	return actual.Message == expected
}
