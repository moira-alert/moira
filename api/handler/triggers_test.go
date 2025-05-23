package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metric_source/remote"

	prometheus "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	dataBase "github.com/moira-alert/moira/database"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"go.uber.org/mock/gomock"

	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	. "github.com/smartystreets/goconvey/convey"
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
	sourceProvider := metricSource.CreateTestMetricSourceProvider(localSource, remoteSource, nil)

	localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
	localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
	fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
	fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

	setValuesToRequestCtx := func(
		ctx context.Context,
		metricSourceProvider *metricSource.SourceProvider,
		limits api.LimitsConfig,
	) context.Context {
		ctx = middleware.SetContextValueForTest(ctx, "metricSourceProvider", metricSourceProvider)
		ctx = middleware.SetContextValueForTest(ctx, "limits", limits)

		return ctx
	}

	Convey("Given a correct payload", t, func() {
		triggerWarnValue := 0.0
		triggerErrorValue := 1.0
		ttlState := moira.TTLState("NODATA")
		triggerDTO := dto.Trigger{
			TriggerModel: dto.TriggerModel{
				ID:          "test_id",
				Name:        "Test trigger",
				Desc:        new(string),
				Targets:     []string{"foo.bar"},
				WarnValue:   &triggerWarnValue,
				ErrorValue:  &triggerErrorValue,
				TriggerType: "rising",
				Tags:        []string{"Normal", "DevOps", "DevOpsGraphite-duty"},
				TTLState:    &ttlState,
				TTL:         0,
				Schedule: &moira.ScheduleData{
					Days: []moira.ScheduleDataDay{
						{
							Name:    "Mon",
							Enabled: true,
						},
					},
				},
				Expression:     "",
				Patterns:       []string{},
				TriggerSource:  moira.GraphiteLocal,
				ClusterId:      moira.DefaultCluster,
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
		request = request.WithContext(setValuesToRequestCtx(request.Context(), sourceProvider, api.GetTestLimitsConfig()))

		triggerDTO.Schedule.Days = moira.GetFilledScheduleDataDays(false)
		triggerDTO.Schedule.Days[0].Enabled = true

		Convey("It should be parsed successfully", func() {
			triggerDTO.TTL = moira.DefaultTTL

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
		request = request.WithContext(setValuesToRequestCtx(request.Context(), sourceProvider, api.GetTestLimitsConfig()))

		Convey("Parser should return en error", func() {
			_, err := getTriggerFromRequest(request)
			So(err, ShouldHaveSameTypeAs, api.ErrorInvalidRequest(fmt.Errorf("")))
		})
	})

	Convey("With incorrect targets errors", t, func() {
		graphiteLocalSrc := mock_metric_source.NewMockMetricSource(mockCtrl)
		graphiteRemoteSrc := mock_metric_source.NewMockMetricSource(mockCtrl)
		prometheusSrc := mock_metric_source.NewMockMetricSource(mockCtrl)
		allSourceProvider := metricSource.CreateTestMetricSourceProvider(graphiteLocalSrc, graphiteRemoteSrc, prometheusSrc)

		graphiteLocalSrc.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
		graphiteRemoteSrc.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
		prometheusSrc.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()

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
				TTL:            moira.DefaultTTL,
				Schedule:       &moira.ScheduleData{},
				Expression:     "",
				Patterns:       []string{},
				ClusterId:      moira.DefaultCluster,
				MuteNewMetrics: false,
				AloneMetrics:   map[string]bool{},
				CreatedAt:      &time.Time{},
				UpdatedAt:      &time.Time{},
				CreatedBy:      "",
				UpdatedBy:      "anonymous",
			},
		}

		Convey("for graphite remote", func() {
			triggerDTO.TriggerSource = moira.GraphiteRemote
			body, _ := json.Marshal(triggerDTO)
			testLogger, _ := logging.GetLogger("Test")

			Convey("when ErrRemoteTriggerResponse returned", func() {
				request := httptest.NewRequest(http.MethodPut, "/trigger", bytes.NewReader(body))
				request.Header.Add("content-type", "application/json")
				request = request.WithContext(setValuesToRequestCtx(request.Context(), allSourceProvider, api.GetTestLimitsConfig()))

				request = middleware.WithLogEntry(request, middleware.NewLogEntry(testLogger, request))

				var returnedErr error = remote.ErrRemoteTriggerResponse{
					InternalError: fmt.Errorf(""),
				}

				graphiteRemoteSrc.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, returnedErr)

				_, errRsp := getTriggerFromRequest(request)
				So(errRsp, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("error from graphite remote: %w", returnedErr)))
			})

			Convey("when ErrRemoteUnavailable", func() {
				request := httptest.NewRequest(http.MethodPut, "/trigger", bytes.NewReader(body))
				request.Header.Add("content-type", "application/json")
				request = request.WithContext(setValuesToRequestCtx(request.Context(), allSourceProvider, api.GetTestLimitsConfig()))

				request = middleware.WithLogEntry(request, middleware.NewLogEntry(testLogger, request))

				var returnedErr error = remote.ErrRemoteUnavailable{
					InternalError: fmt.Errorf(""),
				}

				graphiteRemoteSrc.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, returnedErr)

				_, errRsp := getTriggerFromRequest(request)
				So(errRsp, ShouldResemble, api.ErrorRemoteServerUnavailable(returnedErr))
			})
		})

		Convey("for prometheus remote", func() {
			triggerDTO.TriggerSource = moira.PrometheusRemote
			body, _ := json.Marshal(triggerDTO)

			Convey("with error type = bad_data got bad request", func() {
				request := httptest.NewRequest(http.MethodPut, "/trigger", bytes.NewReader(body))
				request.Header.Add("content-type", "application/json")
				request = request.WithContext(setValuesToRequestCtx(request.Context(), allSourceProvider, api.GetTestLimitsConfig()))

				var returnedErr error = &prometheus.Error{
					Type: prometheus.ErrBadData,
				}

				prometheusSrc.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, returnedErr)

				_, errRsp := getTriggerFromRequest(request)
				So(errRsp, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("invalid prometheus targets: %w", returnedErr)))
			})

			Convey("with other types internal server error is returned", func() {
				otherTypes := []prometheus.ErrorType{
					prometheus.ErrBadResponse,
					prometheus.ErrCanceled,
					prometheus.ErrClient,
					prometheus.ErrExec,
					prometheus.ErrTimeout,
				}

				for _, errType := range otherTypes {
					request := httptest.NewRequest(http.MethodPut, "/trigger", bytes.NewReader(body))
					request.Header.Add("content-type", "application/json")
					request = request.WithContext(setValuesToRequestCtx(request.Context(), allSourceProvider, api.GetTestLimitsConfig()))

					var returnedErr error = &prometheus.Error{
						Type: errType,
					}

					prometheusSrc.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil, returnedErr)

					_, errRsp := getTriggerFromRequest(request)
					So(errRsp, ShouldResemble, api.ErrorInternalServer(returnedErr))
				}
			})
		})
	})
}

func TestGetMetricTTLByTrigger(t *testing.T) {
	request := httptest.NewRequest("", "/", strings.NewReader(""))
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "clustersMetricTTL", MakeTestTTLs()))

	Convey("Given a local trigger", t, func() {
		trigger := dto.Trigger{TriggerModel: dto.TriggerModel{
			TriggerSource: moira.GraphiteLocal,
			ClusterId:     moira.DefaultCluster,
		}}

		Convey("It's metric ttl should be equal to local", func() {
			ttl, err := getMetricTTLByTrigger(request, &trigger)
			So(err, ShouldBeNil)
			So(ttl, ShouldEqual, 65*time.Minute)
		})
	})

	Convey("Given a remote trigger", t, func() {
		trigger := dto.Trigger{TriggerModel: dto.TriggerModel{
			TriggerSource: moira.GraphiteRemote,
			ClusterId:     moira.DefaultCluster,
		}}

		Convey("It's metric ttl should be equal to remote", func() {
			ttl, err := getMetricTTLByTrigger(request, &trigger)
			So(err, ShouldBeNil)
			So(ttl, ShouldEqual, 168*time.Hour)
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
			sourceProvider := metricSource.CreateTestMetricSourceProvider(localSource, remoteSource, nil)

			localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
			localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
			fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

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
					testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "clustersMetricTTL", MakeTestTTLs()))
					testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "limits", api.GetTestLimitsConfig()))

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

	localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()

	fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
	localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
	fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
	fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

	const validateFlag = "validate"

	mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)
	database = mockDb

	mockDb.EXPECT().AcquireTriggerCheckLock(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockDb.EXPECT().DeleteTriggerCheckLock(gomock.Any()).AnyTimes()
	mockDb.EXPECT().GetTriggerLastCheck(gomock.Any()).AnyTimes()
	mockDb.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockDb.EXPECT().SaveTrigger(gomock.Any(), gomock.Any()).AnyTimes()

	Convey("When createTrigger was called with normal input", t, func() {
		urls := []string{
			"/",
			fmt.Sprintf("/trigger?%s", validateFlag),
		}

		Convey("should return RemoteServerUnavailable if remote unavailable, ", func() {
			fetchRemoteErrorTypes := []prometheus.ErrorType{
				// Prometheus error format
				prometheus.ErrServer,
				// VictoriaMetrics error format
				"503",
			}

			for _, fetchRemoteErrorType := range fetchRemoteErrorTypes {
				prometheusRemote := mock_metric_source.NewMockMetricSource(mockCtrl)
				prometheusRemote.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
				prometheusRemote.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &prometheus.Error{Type: fetchRemoteErrorType}).AnyTimes()
				sourceProvider := metricSource.CreateTestMetricSourceProvider(nil, nil, prometheusRemote)

				for _, url := range urls {
					triggerWarnValue := float64(10)
					triggerErrorValue := float64(15)
					triggerDTO := dto.Trigger{
						TriggerModel: dto.TriggerModel{
							Name:          "Test trigger",
							Tags:          []string{"123"},
							WarnValue:     &triggerWarnValue,
							ErrorValue:    &triggerErrorValue,
							Targets:       []string{"my.metric"},
							TriggerSource: moira.PrometheusRemote,
						},
					}
					jsonTrigger, _ := json.Marshal(triggerDTO)
					testRequest := httptest.NewRequest("", url, bytes.NewBuffer(jsonTrigger))
					testRequest.Header.Add("content-type", "application/json")
					testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "metricSourceProvider", sourceProvider))
					testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "clustersMetricTTL", MakeTestTTLs()))
					testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "limits", api.GetTestLimitsConfig()))

					responseWriter := httptest.NewRecorder()
					createTrigger(responseWriter, testRequest)

					Convey(fmt.Sprintf("url=%s, error=%s", url, fetchRemoteErrorType), func() {
						response := responseWriter.Result()
						defer response.Body.Close()
						So(response.StatusCode, ShouldEqual, http.StatusServiceUnavailable)
						So(isTriggerCreated(response), ShouldBeFalse)
					})
				}
			}
		})

		Convey("should return success message, url=", func() {
			sourceProvider := metricSource.CreateTestMetricSourceProvider(localSource, remoteSource, nil)

			for _, url := range urls {
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
				testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "clustersMetricTTL", MakeTestTTLs()))
				testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "limits", api.GetTestLimitsConfig()))

				responseWriter := httptest.NewRecorder()
				createTrigger(responseWriter, testRequest)

				Convey(url, func() {
					response := responseWriter.Result()
					defer response.Body.Close()
					So(response.StatusCode, ShouldEqual, http.StatusOK)
					So(isTriggerCreated(response), ShouldBeTrue)
				})
			}
		})
	})

	Convey("When createTrigger was called with empty targets", t, func() {
		sourceProvider := metricSource.CreateTestMetricSourceProvider(localSource, remoteSource, nil)
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
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "clustersMetricTTL", MakeTestTTLs()))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "limits", api.GetTestLimitsConfig()))

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
		sourceProvider := metricSource.CreateTestMetricSourceProvider(localSource, remoteSource, nil)
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
			request := httptest.NewRequest("", "/", bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "clustersMetricTTL", MakeTestTTLs()))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "limits", api.GetTestLimitsConfig()))

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
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "clustersMetricTTL", MakeTestTTLs()))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "limits", api.GetTestLimitsConfig()))

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
		sourceProvider := metricSource.CreateTestMetricSourceProvider(localSource, remoteSource, nil)
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
			request := httptest.NewRequest("", "/", bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "clustersMetricTTL", MakeTestTTLs()))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "limits", api.GetTestLimitsConfig()))

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
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "clustersMetricTTL", MakeTestTTLs()))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "limits", api.GetTestLimitsConfig()))

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
	defer mockCtrl.Finish()

	localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	prometheusSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	sourceProvider := metricSource.CreateTestMetricSourceProvider(localSource, remoteSource, prometheusSource)

	db := mock_moira_alert.NewMockDatabase(mockCtrl)
	database = db

	defer func() { database = nil }()

	triggerId := "test"
	target := `test_target_value`

	Convey("Given is_remote flag is false and trigger_source is not set", t, func() {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"is_remote": false`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be graphite local", func() {
			setupExpectationsForCreateTrigger(localSource, db, target, triggerId, moira.MakeClusterKey(moira.GraphiteLocal, moira.DefaultCluster))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	Convey("Given is_remote flag is true and trigger_source is not set", t, func() {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"is_remote": true`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be graphite remote", func() {
			setupExpectationsForCreateTrigger(remoteSource, db, target, triggerId, moira.DefaultGraphiteRemoteCluster)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	Convey("Given is_remote flag is not set and trigger_source is graphite_local", t, func() {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"trigger_source": "graphite_local"`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be graphite local", func() {
			setupExpectationsForCreateTrigger(localSource, db, target, triggerId, moira.MakeClusterKey(moira.GraphiteLocal, moira.DefaultCluster))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	Convey("Given is_remote flag is not set and trigger_source is graphite_remote", t, func() {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"trigger_source": "graphite_remote"`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be graphite remote", func() {
			setupExpectationsForCreateTrigger(remoteSource, db, target, triggerId, moira.DefaultGraphiteRemoteCluster)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	Convey("Given is_remote flag is not set and trigger_source is prometheus_remote", t, func() {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"trigger_source": "prometheus_remote"`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be prometheus remote", func() {
			setupExpectationsForCreateTrigger(prometheusSource, db, target, triggerId, moira.MakeClusterKey(moira.PrometheusRemote, moira.DefaultCluster))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	Convey("Given is_remote flag is true and trigger_source is graphite_local", t, func() {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"is_remote": true, "trigger_source": "graphite_local"`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be graphite local", func() {
			setupExpectationsForCreateTrigger(localSource, db, target, triggerId, moira.MakeClusterKey(moira.GraphiteLocal, moira.DefaultCluster))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})
}

func TestTriggersCreatedWithNonDefaultClusterId(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	localSource := mock_metric_source.NewMockMetricSource(mockCtrl)

	remoteStagingCluster := moira.MakeClusterKey(moira.GraphiteLocal, moira.ClusterId("staging"))
	remoteStagingSource := mock_metric_source.NewMockMetricSource(mockCtrl)

	sourceProvider := metricSource.CreateTestMetricSourceProvider(localSource, nil, nil)
	sourceProvider.RegisterSource(remoteStagingCluster, remoteStagingSource)

	db := mock_moira_alert.NewMockDatabase(mockCtrl)
	database = db

	defer func() { database = nil }()

	triggerId := "test"
	target := `test_target_value`

	Convey("Given cluster_id is set", t, func() {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"trigger_source": "graphite_local", "cluster_id": "staging"`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger have non default cluster id", func() {
			setupExpectationsForCreateTrigger(remoteStagingSource, db, target, triggerId, remoteStagingCluster)

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
	clusterKey moira.ClusterKey,
) {
	source.EXPECT().GetMetricsTTLSeconds().Return(int64(3600))
	source.EXPECT().Fetch(target, gomock.Any(), gomock.Any(), gomock.Any()).Return(&local.FetchResult{}, nil)

	db.EXPECT().GetTrigger(triggerId).Return(moira.Trigger{}, dataBase.ErrNil)
	db.EXPECT().AcquireTriggerCheckLock(triggerId, gomock.Any()).Return(nil)
	db.EXPECT().DeleteTriggerCheckLock(triggerId).Return(nil)
	db.EXPECT().GetTriggerLastCheck(triggerId).Return(moira.CheckData{}, dataBase.ErrNil)
	db.EXPECT().SetTriggerLastCheck(triggerId, gomock.Any(), clusterKey).Return(nil)
	db.EXPECT().SaveTrigger(triggerId, gomock.Any()).Return(nil)
}

func newTriggerCreateRequest(
	sourceProvider *metricSource.SourceProvider,
	triggerId string,
	jsonTrigger []byte,
) *http.Request {
	request := httptest.NewRequest(http.MethodPut, "/trigger", bytes.NewBuffer(jsonTrigger))
	request.Header.Add("content-type", "application/json")
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "clustersMetricTTL", MakeTestTTLs()))
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), triggerIDKey, triggerId))
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "limits", api.GetTestLimitsConfig()))

	return request
}

func isTriggerCreated(response *http.Response) bool {
	contentBytes, _ := io.ReadAll(response.Body)
	actual := dto.SaveTriggerResponse{}
	_ = json.Unmarshal(contentBytes, &actual)

	const expected = "trigger created"

	return actual.Message == expected
}

func TestGetTriggerNoisiness(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDB := mock_moira_alert.NewMockDatabase(mockCtrl)
	database = mockDB

	getRequestTriggerNoisiness := func(from, to string) *http.Request {
		request := httptest.NewRequest(http.MethodGet, "/trigger/noisiness", nil)
		request.Header.Add("content-type", "application/json")

		request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "page", int64(0)))
		request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "size", int64(-1)))
		request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "from", from))
		request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "to", to))
		request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "sort", api.AscSortOrder))

		return request
	}

	testTriggerCheck := moira.TriggerCheck{
		Trigger: moira.Trigger{
			ID: "triggerID",
		},
	}

	Convey("Test get trigger noisiness", t, func() {
		now := time.Now()

		from := strconv.FormatInt(now.Add(time.Second*-3).Unix(), 10)
		to := strconv.FormatInt(now.Unix(), 10)

		Convey("with ok", func() {
			responseWriter := httptest.NewRecorder()

			mockDB.EXPECT().GetAllTriggerIDs().Return([]string{testTriggerCheck.ID}, nil)
			mockDB.EXPECT().GetNotificationEventCount(testTriggerCheck.ID, from, to).Return(int64(1))
			mockDB.EXPECT().GetTriggerChecks([]string{testTriggerCheck.ID}).Return([]*moira.TriggerCheck{&testTriggerCheck}, nil)

			getTriggerNoisiness(responseWriter, getRequestTriggerNoisiness(from, to))

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)

			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)

			var gotDTO dto.TriggerNoisinessList
			err = json.Unmarshal(contentBytes, &gotDTO)
			So(err, ShouldBeNil)
			So(&gotDTO, ShouldResemble, &dto.TriggerNoisinessList{
				List: []*dto.TriggerNoisiness{
					{
						Trigger: dto.Trigger{
							TriggerModel: dto.CreateTriggerModel(&testTriggerCheck.Trigger),
						},
						EventsCount: 1,
					},
				},
				Page:  0,
				Size:  -1,
				Total: 1,
			})
		})

		Convey("with error from db", func() {
			responseWriter := httptest.NewRecorder()
			errFromDB := errors.New("some DB error")

			mockDB.EXPECT().GetAllTriggerIDs().Return(nil, errFromDB)

			getTriggerNoisiness(responseWriter, getRequestTriggerNoisiness(from, to))

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)

			contentBytes, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)

			expectedContentBytes, err := json.Marshal(api.ErrorInternalServer(errFromDB))
			So(err, ShouldBeNil)
			So(string(contentBytes), ShouldResemble, string(expectedContentBytes)+"\n")
		})
	})
}

func TestRemoveTriggerHandler(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)
	database = mockDb
	triggerID := "my-trigger-id"
	adminLogin := "admin"
	userLogin := "user"
	ownerLogin := "owner"

	Convey("When auth is false", t, func() {
		auth := api.Authorization{
			Enabled: true,
			AdminList: map[string]struct{}{
				adminLogin: {},
			},
			LimitedChangeTriggerOwners: map[string]struct{}{
				ownerLogin: {},
			},
		}
		logger, _ := logging.GetLogger("Test")
		config := &api.Config{Authorization: auth}
		webConfig := &api.WebConfig{
			SupportEmail: "test",
			Contacts:     []api.WebContact{},
		}
		trigger := moira.Trigger{
			CreatedBy: ownerLogin,
		}

		Convey("And when success from DB, should return success", func() {
			mockDb.EXPECT().RemoveTrigger(triggerID).Return(nil)
			mockDb.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
			mockDb.EXPECT().GetTriggerThrottling(triggerID)

			handler := NewHandler(mockDb, logger, nil, config, nil, webConfig, nil)

			responseWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest(http.MethodDelete, "/api/trigger/"+triggerID, strings.NewReader(""))
			testRequest.Header.Add("x-webauth-user", adminLogin)
			testRequest.Header.Add("content-type", "application/json")
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(context.Background(), "auth", auth))
			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("And when error while removing from DB, should return error", func() {
			mockDb.EXPECT().RemoveTrigger(triggerID).Return(errors.New("error"))
			mockDb.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
			mockDb.EXPECT().GetTriggerThrottling(triggerID)

			handler := NewHandler(mockDb, logger, nil, config, nil, webConfig, nil)

			responseWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest(http.MethodDelete, "/api/trigger/"+triggerID, strings.NewReader(""))
			testRequest.Header.Add("x-webauth-user", adminLogin)
			testRequest.Header.Add("content-type", "application/json")
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(context.Background(), "auth", auth))
			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})
	})

	Convey("When auth is true, trigger owner is not admin", t, func() {
		auth := api.Authorization{
			Enabled: true,
			AdminList: map[string]struct{}{
				adminLogin: {},
			},
			LimitedChangeTriggerOwners: map[string]struct{}{
				ownerLogin: {},
			},
		}
		logger, _ := logging.GetLogger("Test")
		config := &api.Config{Authorization: auth}
		webConfig := &api.WebConfig{
			SupportEmail: "test",
			Contacts:     []api.WebContact{},
		}
		trigger := moira.Trigger{
			CreatedBy: ownerLogin,
		}

		Convey("When request from moira-admin, should be ok", func() {
			mockDb.EXPECT().RemoveTrigger(triggerID).Return(nil)
			mockDb.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
			mockDb.EXPECT().GetTriggerThrottling(triggerID)

			handler := NewHandler(mockDb, logger, nil, config, nil, webConfig, nil)

			responseWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest(http.MethodDelete, "/api/trigger/"+triggerID, strings.NewReader(""))
			testRequest.Header.Add("x-webauth-user", adminLogin)
			testRequest.Header.Add("content-type", "application/json")
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(context.Background(), "auth", auth))
			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("When request from trigger-owner, should be ok", func() {
			mockDb.EXPECT().RemoveTrigger(triggerID).Return(nil)
			mockDb.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
			mockDb.EXPECT().GetTriggerThrottling(triggerID)

			handler := NewHandler(mockDb, logger, nil, config, nil, webConfig, nil)

			responseWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest(http.MethodDelete, "/api/trigger/"+triggerID, strings.NewReader(""))
			testRequest.Header.Add("x-webauth-user", ownerLogin)
			testRequest.Header.Add("content-type", "application/json")
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(context.Background(), "auth", auth))
			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("When request from other-user, should be forbidden", func() {
			mockDb.EXPECT().GetTrigger(triggerID).Return(trigger, nil)
			mockDb.EXPECT().GetTriggerThrottling(triggerID)

			handler := NewHandler(mockDb, logger, nil, config, nil, webConfig, nil)

			responseWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest(http.MethodDelete, "/api/trigger/"+triggerID, strings.NewReader(""))
			testRequest.Header.Add("x-webauth-user", userLogin)
			testRequest.Header.Add("content-type", "application/json")
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(context.Background(), "auth", auth))
			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusForbidden)
		})
	})
}

func TestUpdateTriggerHandler(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)
	database = mockDb
	triggerID := "my-trigger-id"
	adminLogin := "admin"
	userLogin := "user"
	ownerLogin := "owner"
	warnValue := float64(4)

	localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	sourceProvider := metricSource.CreateTestMetricSourceProvider(localSource, remoteSource, nil)

	localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()

	fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
	fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).Times(1)
	localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
	fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
	fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

	Convey("When auth is false", t, func() {
		auth := api.Authorization{
			Enabled: true,
			AdminList: map[string]struct{}{
				adminLogin: {},
			},
			LimitedChangeTriggerOwners: map[string]struct{}{
				ownerLogin: {},
			},
		}
		logger, _ := logging.GetLogger("Test")
		config := &api.Config{
			Authorization: auth,
			Limits: api.LimitsConfig{
				Trigger: api.TriggerLimits{
					MaxNameSize: 100,
				},
			},
		}
		webConfig := &api.WebConfig{
			SupportEmail: "test",
			Contacts:     []api.WebContact{},
		}
		trigger := moira.Trigger{
			Targets: []string{
				"foo.bar",
			},
			Tags: []string{
				"tag1",
			},
			Name:        "Not enough disk space left",
			ID:          triggerID,
			CreatedBy:   ownerLogin,
			WarnValue:   &warnValue,
			TriggerType: "rising",
		}

		jsonTrigger, err := json.Marshal(trigger)
		So(err, ShouldBeNil)

		Convey("And when success from DB, should return success", func() {
			mockDb.EXPECT().GetTrigger(triggerID).Return(trigger, nil).AnyTimes()
			mockDb.EXPECT().AcquireTriggerCheckLock(triggerID, 30).Return(nil)
			mockDb.EXPECT().DeleteTriggerCheckLock(triggerID)
			mockDb.EXPECT().GetTriggerThrottling(triggerID)
			mockDb.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(moira.CheckData{}, dataBase.ErrNil)
			mockDb.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockDb.EXPECT().SaveTrigger(gomock.Any(), gomock.Any()).Return(nil)

			handler := NewHandler(mockDb, logger, nil, config, sourceProvider, webConfig, nil)

			responseWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest(http.MethodPut, "/api/trigger/"+triggerID, bytes.NewBuffer(jsonTrigger))
			testRequest.Header.Add("x-webauth-user", adminLogin)
			testRequest.Header.Add("content-type", "application/json")
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(context.Background(), "auth", auth))
			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("And when error while save from DB, should return error", func() {
			mockDb.EXPECT().GetTrigger(triggerID).Return(trigger, nil).AnyTimes()
			mockDb.EXPECT().AcquireTriggerCheckLock(triggerID, 30).Return(nil)
			mockDb.EXPECT().DeleteTriggerCheckLock(triggerID)
			mockDb.EXPECT().GetTriggerThrottling(triggerID)
			mockDb.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(moira.CheckData{}, dataBase.ErrNil)
			mockDb.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockDb.EXPECT().SaveTrigger(gomock.Any(), gomock.Any()).Return(errors.New("error"))

			handler := NewHandler(mockDb, logger, nil, config, sourceProvider, webConfig, nil)

			responseWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest(http.MethodPut, "/api/trigger/"+triggerID, bytes.NewBuffer(jsonTrigger))
			testRequest.Header.Add("x-webauth-user", adminLogin)
			testRequest.Header.Add("content-type", "application/json")
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(context.Background(), "auth", auth))
			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})
	})

	Convey("When auth is true, trigger owner is not admin", t, func() {
		auth := api.Authorization{
			Enabled: true,
			AdminList: map[string]struct{}{
				adminLogin: {},
			},
			LimitedChangeTriggerOwners: map[string]struct{}{
				ownerLogin: {},
			},
		}
		logger, _ := logging.GetLogger("Test")
		config := &api.Config{
			Authorization: auth,
			Limits: api.LimitsConfig{
				Trigger: api.TriggerLimits{
					MaxNameSize: 100,
				},
			},
		}
		webConfig := &api.WebConfig{
			SupportEmail: "test",
			Contacts:     []api.WebContact{},
		}
		trigger := moira.Trigger{
			Targets: []string{
				"foo.bar",
			},
			Tags: []string{
				"tag1",
			},
			Name:        "Not enough disk space left",
			ID:          triggerID,
			CreatedBy:   ownerLogin,
			WarnValue:   &warnValue,
			TriggerType: "rising",
		}

		jsonTrigger, err := json.Marshal(trigger)
		So(err, ShouldBeNil)

		Convey("When request from moira-admin, should be ok", func() {
			mockDb.EXPECT().GetTrigger(triggerID).Return(trigger, nil).AnyTimes()
			mockDb.EXPECT().AcquireTriggerCheckLock(triggerID, 30).Return(nil)
			mockDb.EXPECT().DeleteTriggerCheckLock(triggerID)
			mockDb.EXPECT().GetTriggerThrottling(triggerID)
			mockDb.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(moira.CheckData{}, dataBase.ErrNil)
			mockDb.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockDb.EXPECT().SaveTrigger(gomock.Any(), gomock.Any()).Return(nil)

			handler := NewHandler(mockDb, logger, nil, config, sourceProvider, webConfig, nil)

			responseWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest(http.MethodPut, "/api/trigger/"+triggerID, bytes.NewBuffer(jsonTrigger))
			testRequest.Header.Add("x-webauth-user", adminLogin)
			testRequest.Header.Add("content-type", "application/json")
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(context.Background(), "auth", auth))
			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("When request from trigger-owner, should be ok", func() {
			mockDb.EXPECT().GetTrigger(triggerID).Return(trigger, nil).AnyTimes()
			mockDb.EXPECT().AcquireTriggerCheckLock(triggerID, 30).Return(nil)
			mockDb.EXPECT().DeleteTriggerCheckLock(triggerID)
			mockDb.EXPECT().GetTriggerThrottling(triggerID)
			mockDb.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(moira.CheckData{}, dataBase.ErrNil)
			mockDb.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockDb.EXPECT().SaveTrigger(gomock.Any(), gomock.Any()).Return(nil)

			handler := NewHandler(mockDb, logger, nil, config, sourceProvider, webConfig, nil)

			responseWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest(http.MethodPut, "/api/trigger/"+triggerID, bytes.NewBuffer(jsonTrigger))
			testRequest.Header.Add("x-webauth-user", ownerLogin)
			testRequest.Header.Add("content-type", "application/json")
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(context.Background(), "auth", auth))
			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("When request from other-user, should be forbidden", func() {
			mockDb.EXPECT().GetTrigger(triggerID).Return(trigger, nil).AnyTimes()
			mockDb.EXPECT().GetTriggerThrottling(triggerID)

			handler := NewHandler(mockDb, logger, nil, config, sourceProvider, webConfig, nil)

			responseWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest(http.MethodPut, "/api/trigger/"+triggerID, bytes.NewBuffer(jsonTrigger))
			testRequest.Header.Add("x-webauth-user", userLogin)
			testRequest.Header.Add("content-type", "application/json")
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(context.Background(), "auth", auth))
			handler.ServeHTTP(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			So(response.StatusCode, ShouldEqual, http.StatusForbidden)
		})
	})
}
