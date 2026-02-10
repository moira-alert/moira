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
	"github.com/stretchr/testify/require"

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
)

func TestGetSearchRequestString(t *testing.T) {
	t.Run("Given a search request string", func(t *testing.T) {
		t.Run("The value should be converted into lower case", func(t *testing.T) {
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
				require.Equal(t, testCase.expectedSearchRequest, searchRequest)
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

	t.Run("Given a correct payload", func(t *testing.T) {
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

		t.Run("It should be parsed successfully", func(t *testing.T) {
			triggerDTO.TTL = moira.DefaultTTL

			trigger, err := getTriggerFromRequest(request)

			require.Nil(t, err)
			require.Equal(t, trigger, &triggerDTO)
		})
	})

	t.Run("Given an incorrect payload", func(t *testing.T) {
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

		t.Run("Parser should return en error", func(t *testing.T) {
			_, err := getTriggerFromRequest(request)
			require.IsType(t, api.ErrorInvalidRequest(fmt.Errorf("")), err)
		})
	})

	t.Run("With incorrect targets errors", func(t *testing.T) {
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

		t.Run("for graphite remote", func(t *testing.T) {
			triggerDTO.TriggerSource = moira.GraphiteRemote
			body, _ := json.Marshal(triggerDTO)
			testLogger, _ := logging.GetLogger("Test")

			t.Run("when ErrRemoteTriggerResponse returned", func(t *testing.T) {
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
				require.Equal(t, errRsp, api.ErrorInvalidRequest(fmt.Errorf("error from graphite remote: %w", returnedErr)))
			})

			t.Run("when ErrRemoteUnavailable", func(t *testing.T) {
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
				require.Equal(t, errRsp, api.ErrorRemoteServerUnavailable(returnedErr))
			})
		})

		t.Run("for prometheus remote", func(t *testing.T) {
			triggerDTO.TriggerSource = moira.PrometheusRemote
			body, _ := json.Marshal(triggerDTO)

			t.Run("with error type = bad_data got bad request", func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPut, "/trigger", bytes.NewReader(body))
				request.Header.Add("content-type", "application/json")
				request = request.WithContext(setValuesToRequestCtx(request.Context(), allSourceProvider, api.GetTestLimitsConfig()))

				var returnedErr error = &prometheus.Error{
					Type: prometheus.ErrBadData,
				}

				prometheusSrc.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, returnedErr)

				_, errRsp := getTriggerFromRequest(request)
				require.Equal(t, errRsp, api.ErrorInvalidRequest(fmt.Errorf("invalid prometheus targets: %w", returnedErr)))
			})

			t.Run("with other types internal server error is returned", func(t *testing.T) {
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
					require.Equal(t, errRsp, api.ErrorInternalServer(returnedErr))
				}
			})
		})
	})
}

func TestGetMetricTTLByTrigger(t *testing.T) {
	request := httptest.NewRequest("", "/", strings.NewReader(""))
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "clustersMetricTTL", MakeTestTTLs()))

	t.Run("Given a local trigger", func(t *testing.T) {
		trigger := dto.Trigger{TriggerModel: dto.TriggerModel{
			TriggerSource: moira.GraphiteLocal,
			ClusterId:     moira.DefaultCluster,
		}}

		t.Run("It's metric ttl should be equal to local", func(t *testing.T) {
			ttl, err := getMetricTTLByTrigger(request, &trigger)
			require.NoError(t, err)
			require.Equal(t, 65*time.Minute, ttl)
		})
	})

	t.Run("Given a remote trigger", func(t *testing.T) {
		trigger := dto.Trigger{TriggerModel: dto.TriggerModel{
			TriggerSource: moira.GraphiteRemote,
			ClusterId:     moira.DefaultCluster,
		}}

		t.Run("It's metric ttl should be equal to remote", func(t *testing.T) {
			ttl, err := getMetricTTLByTrigger(request, &trigger)
			require.NoError(t, err)
			require.Equal(t, 168*time.Hour, ttl)
		})
	})
}

func TestTriggerCheckHandler(t *testing.T) {
	t.Run("Test triggerCheck handler", func(t *testing.T) {
		t.Run("Checking target metric ttl validation", func(t *testing.T) {
			mockCtrl := gomock.NewController(t)

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
				t.Run(fmt.Sprintf("TestCase #%d", n), func(t *testing.T) {
					responseWriter := httptest.NewRecorder()
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

					require.Equal(t, testCase.expectedResponse, contents)
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

	t.Run("When createTrigger was called with normal input", func(t *testing.T) {
		urls := []string{
			"/",
			fmt.Sprintf("/trigger?%s", validateFlag),
		}

		t.Run("should return RemoteServerUnavailable if remote unavailable, ", func(t *testing.T) {
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

					t.Run(fmt.Sprintf("url=%s, error=%s", url, fetchRemoteErrorType), func(t *testing.T) {
						response := responseWriter.Result()
						defer response.Body.Close()

						require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
						require.False(t, isTriggerCreated(response))
					})
				}
			}
		})

		t.Run("should return success message, url=", func(t *testing.T) {
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

				t.Run(url, func(t *testing.T) {
					response := responseWriter.Result()
					defer response.Body.Close()

					require.Equal(t, http.StatusOK, response.StatusCode)
					require.True(t, isTriggerCreated(response))
				})
			}
		})
	})

	t.Run("When createTrigger was called with empty targets", func(t *testing.T) {
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

			t.Run(fmt.Sprintf("should return 400, url=%s", url), func(t *testing.T) {
				response := responseWriter.Result()
				defer response.Body.Close()

				require.Equal(t, http.StatusBadRequest, response.StatusCode)
			})
		}
	})

	t.Run("When createTrigger was called with target with warning function", func(t *testing.T) {
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

		t.Run("without validate like before", func(t *testing.T) {
			request := httptest.NewRequest("", "/", bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "clustersMetricTTL", MakeTestTTLs()))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "limits", api.GetTestLimitsConfig()))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			t.Run("should return 200", func(t *testing.T) {
				response := responseWriter.Result()
				defer response.Body.Close()

				require.Equal(t, http.StatusOK, response.StatusCode)
				require.True(t, isTriggerCreated(response))
			})
		})

		t.Run("with validate", func(t *testing.T) {
			request := httptest.NewRequest("", fmt.Sprintf("/trigger?%s", validateFlag), bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "clustersMetricTTL", MakeTestTTLs()))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "limits", api.GetTestLimitsConfig()))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			t.Run("should return 200 and tree of problems", func(t *testing.T) {
				response := responseWriter.Result()
				defer response.Body.Close()

				require.Equal(t, http.StatusOK, response.StatusCode)

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
				require.Equal(t, expectedTargets, actual.CheckResult.Targets)

				const expected = "trigger created"

				require.Equal(t, expected, actual.Message)
			})
		})
	})

	t.Run("When createTrigger was called with target with bad (error) function", func(t *testing.T) {
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

		t.Run("without validate like before", func(t *testing.T) {
			request := httptest.NewRequest("", "/", bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "clustersMetricTTL", MakeTestTTLs()))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "limits", api.GetTestLimitsConfig()))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			t.Run("should return 200", func(t *testing.T) {
				response := responseWriter.Result()
				defer response.Body.Close()

				require.Equal(t, http.StatusOK, response.StatusCode)
				require.True(t, isTriggerCreated(response))
			})
		})

		t.Run("with validate", func(t *testing.T) {
			request := httptest.NewRequest("", fmt.Sprintf("/trigger?%s", validateFlag), bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "clustersMetricTTL", MakeTestTTLs()))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "limits", api.GetTestLimitsConfig()))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			t.Run("should return 400 and tree of problems", func(t *testing.T) {
				response := responseWriter.Result()
				defer response.Body.Close()

				require.Equal(t, "application/json; charset=utf-8", response.Header.Get("Content-Type"))
				require.Equal(t, http.StatusTeapot, response.StatusCode)

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
				require.Equal(t, expected, actual)
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

	t.Run("Given is_remote flag is false and trigger_source is not set", func(t *testing.T) {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"is_remote": false`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		t.Run("Expect trigger to be graphite local", func(t *testing.T) {
			setupExpectationsForCreateTrigger(localSource, db, target, triggerId, moira.MakeClusterKey(moira.GraphiteLocal, moira.DefaultCluster))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			require.Equal(t, 200, responseWriter.Code)
		})
	})

	t.Run("Given is_remote flag is true and trigger_source is not set", func(t *testing.T) {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"is_remote": true`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		t.Run("Expect trigger to be graphite remote", func(t *testing.T) {
			setupExpectationsForCreateTrigger(remoteSource, db, target, triggerId, moira.DefaultGraphiteRemoteCluster)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			require.Equal(t, 200, responseWriter.Code)
		})
	})

	t.Run("Given is_remote flag is not set and trigger_source is graphite_local", func(t *testing.T) {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"trigger_source": "graphite_local"`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		t.Run("Expect trigger to be graphite local", func(t *testing.T) {
			setupExpectationsForCreateTrigger(localSource, db, target, triggerId, moira.MakeClusterKey(moira.GraphiteLocal, moira.DefaultCluster))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			require.Equal(t, 200, responseWriter.Code)
		})
	})

	t.Run("Given is_remote flag is not set and trigger_source is graphite_remote", func(t *testing.T) {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"trigger_source": "graphite_remote"`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		t.Run("Expect trigger to be graphite remote", func(t *testing.T) {
			setupExpectationsForCreateTrigger(remoteSource, db, target, triggerId, moira.DefaultGraphiteRemoteCluster)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			require.Equal(t, 200, responseWriter.Code)
		})
	})

	t.Run("Given is_remote flag is not set and trigger_source is prometheus_remote", func(t *testing.T) {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"trigger_source": "prometheus_remote"`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		t.Run("Expect trigger to be prometheus remote", func(t *testing.T) {
			setupExpectationsForCreateTrigger(prometheusSource, db, target, triggerId, moira.MakeClusterKey(moira.PrometheusRemote, moira.DefaultCluster))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			require.Equal(t, 200, responseWriter.Code)
		})
	})

	t.Run("Given is_remote flag is true and trigger_source is graphite_local", func(t *testing.T) {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"is_remote": true, "trigger_source": "graphite_local"`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		t.Run("Expect trigger to be graphite local", func(t *testing.T) {
			setupExpectationsForCreateTrigger(localSource, db, target, triggerId, moira.MakeClusterKey(moira.GraphiteLocal, moira.DefaultCluster))

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			require.Equal(t, 200, responseWriter.Code)
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

	t.Run("Given cluster_id is set", func(t *testing.T) {
		jsonTrigger := makeTestTriggerJson(target, triggerId, `"trigger_source": "graphite_local", "cluster_id": "staging"`)
		request := newTriggerCreateRequest(sourceProvider, triggerId, jsonTrigger)

		t.Run("Expect trigger have non default cluster id", func(t *testing.T) {
			setupExpectationsForCreateTrigger(remoteStagingSource, db, target, triggerId, remoteStagingCluster)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			require.Equal(t, 200, responseWriter.Code)
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

	t.Run("Test get trigger noisiness", func(t *testing.T) {
		now := time.Now()

		from := strconv.FormatInt(now.Add(time.Second*-3).Unix(), 10)
		to := strconv.FormatInt(now.Unix(), 10)

		t.Run("with ok", func(t *testing.T) {
			responseWriter := httptest.NewRecorder()

			mockDB.EXPECT().GetAllTriggerIDs().Return([]string{testTriggerCheck.ID}, nil)
			mockDB.EXPECT().GetNotificationEventCount(testTriggerCheck.ID, from, to).Return(int64(1))
			mockDB.EXPECT().GetTriggerChecks([]string{testTriggerCheck.ID}).Return([]*moira.TriggerCheck{&testTriggerCheck}, nil)

			getTriggerNoisiness(responseWriter, getRequestTriggerNoisiness(from, to))

			response := responseWriter.Result()
			defer response.Body.Close()

			require.Equal(t, http.StatusOK, response.StatusCode)

			contentBytes, err := io.ReadAll(response.Body)
			require.NoError(t, err)

			var gotDTO dto.TriggerNoisinessList

			err = json.Unmarshal(contentBytes, &gotDTO)
			require.NoError(t, err)
			require.Equal(t, &dto.TriggerNoisinessList{
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
			}, &gotDTO)
		})

		t.Run("with error from db", func(t *testing.T) {
			responseWriter := httptest.NewRecorder()
			errFromDB := errors.New("some DB error")

			mockDB.EXPECT().GetAllTriggerIDs().Return(nil, errFromDB)

			getTriggerNoisiness(responseWriter, getRequestTriggerNoisiness(from, to))

			response := responseWriter.Result()
			defer response.Body.Close()

			require.Equal(t, http.StatusInternalServerError, response.StatusCode)

			contentBytes, err := io.ReadAll(response.Body)
			require.NoError(t, err)

			expectedContentBytes, err := json.Marshal(api.ErrorInternalServer(errFromDB))
			require.NoError(t, err)
			require.Equal(t, string(contentBytes), string(expectedContentBytes)+"\n")
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

	t.Run("When auth is false", func(t *testing.T) {
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

		t.Run("And when success from DB, should return success", func(t *testing.T) {
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

			require.Equal(t, http.StatusOK, response.StatusCode)
		})

		t.Run("And when error while removing from DB, should return error", func(t *testing.T) {
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

			require.Equal(t, http.StatusInternalServerError, response.StatusCode)
		})
	})

	t.Run("When auth is true, trigger owner is not admin", func(t *testing.T) {
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

		t.Run("When request from moira-admin, should be ok", func(t *testing.T) {
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

			require.Equal(t, http.StatusOK, response.StatusCode)
		})

		t.Run("When request from trigger-owner, should be ok", func(t *testing.T) {
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

			require.Equal(t, http.StatusOK, response.StatusCode)
		})

		t.Run("When request from other-user, should be forbidden", func(t *testing.T) {
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

			require.Equal(t, http.StatusForbidden, response.StatusCode)
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

	t.Run("When auth is false", func(t *testing.T) {
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
		require.NoError(t, err)

		t.Run("And when success from DB, should return success", func(t *testing.T) {
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

			require.Equal(t, http.StatusOK, response.StatusCode)
		})

		t.Run("And when error while save from DB, should return error", func(t *testing.T) {
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

			require.Equal(t, http.StatusInternalServerError, response.StatusCode)
		})
	})

	t.Run("When auth is true, trigger owner is not admin", func(t *testing.T) {
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
		require.NoError(t, err)

		t.Run("When request from moira-admin, should be ok", func(t *testing.T) {
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

			require.Equal(t, http.StatusOK, response.StatusCode)
		})

		t.Run("When request from trigger-owner, should be ok", func(t *testing.T) {
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

			require.Equal(t, http.StatusOK, response.StatusCode)
		})

		t.Run("When request from other-user, should be forbidden", func(t *testing.T) {
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

			require.Equal(t, http.StatusForbidden, response.StatusCode)
		})
	})
}
