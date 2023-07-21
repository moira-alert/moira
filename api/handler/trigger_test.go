package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/xiam/to"

	dataBase "github.com/moira-alert/moira/database"
)

func TestGetTrigger(t *testing.T) {
	Convey("Get trigger by id", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		responseWriter := httptest.NewRecorder()
		mockDb := mock_moira_alert.NewMockDatabase(mockCtrl)

		Convey("When success and have empty created_at & updated_at should return null in response", func() {
			throttlingTime := time.Date(2022, time.June, 7, 10, 0, 0, 0, time.UTC)
			mockDb.EXPECT().GetTrigger("triggerID-0000000000001").Return(moira.Trigger{
				ID:            "triggerID-0000000000001",
				TriggerSource: moira.GraphiteLocal,
			}, nil)
			mockDb.EXPECT().GetTriggerThrottling("triggerID-0000000000001").Return(throttlingTime, throttlingTime)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodGet, "/trigger/triggerID-0000000000001", nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "triggerID", "triggerID-0000000000001"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "populated", false))
			testRequest.Header.Add("content-type", "application/json")

			getTrigger(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := "{\"id\":\"triggerID-0000000000001\",\"name\":\"\",\"targets\":null,\"warn_value\":null,\"error_value\":null,\"trigger_type\":\"\",\"tags\":null,\"expression\":\"\",\"patterns\":null,\"is_remote\":false,\"trigger_source\":\"graphite_local\",\"mute_new_metrics\":false,\"alone_metrics\":null,\"created_at\":null,\"updated_at\":null,\"created_by\":\"\",\"updated_by\":\"\",\"throttling\":0}\n"
			So(contents, ShouldEqual, expected)
		})

		Convey("When success and have not empty created_at & updated_at should return datetime in response", func() {
			throttlingTime := time.Date(2022, time.June, 7, 10, 0, 0, 0, time.UTC)
			triggerTime := time.Date(2022, time.June, 7, 10, 0, 0, 0, time.UTC).Unix()
			mockDb.EXPECT().GetTrigger("triggerID-0000000000001").
				Return(
					moira.Trigger{
						ID:            "triggerID-0000000000001",
						CreatedAt:     &triggerTime,
						TriggerSource: moira.GraphiteLocal,
						UpdatedAt:     &triggerTime,
					},
					nil)
			mockDb.EXPECT().GetTriggerThrottling("triggerID-0000000000001").Return(throttlingTime, throttlingTime)
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodGet, "/trigger/triggerID-0000000000001", nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "triggerID", "triggerID-0000000000001"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "populated", false))
			testRequest.Header.Add("content-type", "application/json")

			getTrigger(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := "{\"id\":\"triggerID-0000000000001\",\"name\":\"\",\"targets\":null,\"warn_value\":null,\"error_value\":null,\"trigger_type\":\"\",\"tags\":null,\"expression\":\"\",\"patterns\":null,\"is_remote\":false,\"trigger_source\":\"graphite_local\",\"mute_new_metrics\":false,\"alone_metrics\":null,\"created_at\":\"2022-06-07T10:00:00Z\",\"updated_at\":\"2022-06-07T10:00:00Z\",\"created_by\":\"\",\"updated_by\":\"\",\"throttling\":0}\n"
			So(contents, ShouldEqual, expected)
		})

		Convey("When cannot get trigger should have error in response", func() {
			mockDb.EXPECT().GetTrigger("triggerID-0000000000001").Return(moira.Trigger{}, fmt.Errorf("cannot get trigger"))
			database = mockDb

			testRequest := httptest.NewRequest(http.MethodGet, "/trigger/triggerID-0000000000001", nil)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "triggerID", "triggerID-0000000000001"))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "populated", false))
			testRequest.Header.Add("content-type", "application/json")

			getTrigger(responseWriter, testRequest)

			response := responseWriter.Result()
			defer response.Body.Close()

			contentBytes, _ := io.ReadAll(response.Body)
			contents := string(contentBytes)
			expected := "{\"status\":\"Internal Server Error\",\"error\":\"cannot get trigger\"}\n"
			So(contents, ShouldEqual, expected)
		})
	})
}

func TestUpdateTrigger(t *testing.T) {
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

	const triggerIDKey = "triggerID"
	const triggerID = "test"

	Convey("When updateTrigger was called with normal input", t, func() {
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
			trigger := moira.Trigger{
				Name:       "Test trigger",
				Tags:       []string{"123"},
				WarnValue:  &triggerWarnValue,
				ErrorValue: &triggerErrorValue,
				Targets:    []string{"my.metric"},
				/// IsRemote:   false,
				TriggerSource: moira.GraphiteLocal,
			}
			mockDb.EXPECT().GetTrigger(gomock.Any()).Return(trigger, nil)

			jsonTrigger, _ := json.Marshal(trigger)
			testRequest := httptest.NewRequest("", url, bytes.NewBuffer(jsonTrigger))
			testRequest.Header.Add("content-type", "application/json")
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "metricSourceProvider", sourceProvider))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "localMetricTTL", to.Duration("65m")))

			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), triggerIDKey, triggerID))

			responseWriter := httptest.NewRecorder()
			updateTrigger(responseWriter, testRequest)

			Convey(fmt.Sprintf("should return success message, url=%s", url), func() {
				response := responseWriter.Result()
				defer response.Body.Close()

				So(response.StatusCode, ShouldEqual, http.StatusOK)
				So(isTriggerUpdated(response), ShouldBeTrue)
			})
		}
	})

	Convey("When updateTrigger was called with empty targets", t, func() {
		urls := []string{
			"/",
			fmt.Sprintf("/trigger?%s", validateFlag),
		}

		for _, url := range urls {
			triggerWarnValue := float64(10)
			triggerErrorValue := float64(15)
			trigger := dto.Trigger{
				TriggerModel: dto.TriggerModel{
					Name:          "Test trigger",
					Tags:          []string{"123"},
					WarnValue:     &triggerWarnValue,
					ErrorValue:    &triggerErrorValue,
					Targets:       []string{},
					TriggerSource: moira.GraphiteLocal,
				},
			}

			jsonTrigger, _ := json.Marshal(trigger)
			request := httptest.NewRequest("", url, bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "localMetricTTL", to.Duration("65m")))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), triggerIDKey, triggerID))

			responseWriter := httptest.NewRecorder()
			updateTrigger(responseWriter, request)

			Convey(fmt.Sprintf("should return 400, url=%s", url), func() {
				response := responseWriter.Result()
				defer response.Body.Close()
				So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
			})
		}
	})

	Convey("When updateTrigger was called with target with warning function", t, func() {
		triggerWarnValue := float64(10)
		triggerErrorValue := float64(15)
		trigger := moira.Trigger{
			Name:       "Test trigger",
			Tags:       []string{"123"},
			WarnValue:  &triggerWarnValue,
			ErrorValue: &triggerErrorValue,
			Targets:    []string{"alias(consolidateBy(Sales.widgets.largeBlue, 'sum'), 'alias to test nesting')"},
			/// IsRemote:   false,
			TriggerSource: moira.GraphiteLocal,
		}

		jsonTrigger, _ := json.Marshal(trigger)

		Convey("without validate like before", func() {
			mockDb.EXPECT().GetTrigger(gomock.Any()).Return(trigger, nil)
			mockDb.EXPECT().AcquireTriggerCheckLock(gomock.Any(), gomock.Any()).Return(nil)
			mockDb.EXPECT().DeleteTriggerCheckLock(gomock.Any())
			mockDb.EXPECT().GetTriggerLastCheck(gomock.Any())
			mockDb.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any())
			mockDb.EXPECT().SaveTrigger(gomock.Any(), gomock.Any())

			request := httptest.NewRequest("", "/", bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "localMetricTTL", to.Duration("65m")))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), triggerIDKey, triggerID))

			responseWriter := httptest.NewRecorder()
			updateTrigger(responseWriter, request)

			Convey("should return 200", func() {
				response := responseWriter.Result()
				defer response.Body.Close()
				So(response.StatusCode, ShouldEqual, http.StatusOK)
				So(isTriggerUpdated(response), ShouldBeTrue)
			})
		})

		Convey("with validate", func() {
			mockDb.EXPECT().GetTrigger(gomock.Any()).Return(trigger, nil)
			mockDb.EXPECT().AcquireTriggerCheckLock(gomock.Any(), gomock.Any()).Return(nil)
			mockDb.EXPECT().DeleteTriggerCheckLock(gomock.Any())
			mockDb.EXPECT().GetTriggerLastCheck(gomock.Any())
			mockDb.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any())
			mockDb.EXPECT().SaveTrigger(gomock.Any(), gomock.Any())

			request := httptest.NewRequest("", fmt.Sprintf("/trigger?%s", validateFlag), bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "localMetricTTL", to.Duration("65m")))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), triggerIDKey, triggerID))

			responseWriter := httptest.NewRecorder()
			updateTrigger(responseWriter, request)

			Convey("should return 200 and tree of problems", func() {
				response := responseWriter.Result()
				defer response.Body.Close()

				So(response.StatusCode, ShouldEqual, http.StatusOK)

				contentBytes, _ := io.ReadAll(response.Body)
				actual := dto.SaveTriggerResponse{}
				_ = json.Unmarshal(contentBytes, &actual)

				So(actual.ID, ShouldEqual, triggerID)
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
				const expected = "trigger updated"
				So(actual.Message, ShouldEqual, expected)
			})
		})
	})

	Convey("When updateTrigger was called with target with bad (error) function", t, func() {
		triggerWarnValue := float64(10)
		triggerErrorValue := float64(15)
		trigger := moira.Trigger{
			Name:       "Test trigger",
			Tags:       []string{"123"},
			WarnValue:  &triggerWarnValue,
			ErrorValue: &triggerErrorValue,
			Targets:    []string{"alias(summarize(my.metric, '5min'), 'alias to test nesting')"},
			/// IsRemote:   false,
			TriggerSource: moira.GraphiteLocal,
		}
		jsonTrigger, _ := json.Marshal(trigger)

		Convey("without validate like before", func() {
			mockDb.EXPECT().GetTrigger(gomock.Any()).Return(trigger, nil)
			mockDb.EXPECT().AcquireTriggerCheckLock(gomock.Any(), gomock.Any()).Return(nil)
			mockDb.EXPECT().DeleteTriggerCheckLock(gomock.Any())
			mockDb.EXPECT().GetTriggerLastCheck(gomock.Any())
			mockDb.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any())
			mockDb.EXPECT().SaveTrigger(gomock.Any(), gomock.Any())

			request := httptest.NewRequest("", "/", bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "localMetricTTL", to.Duration("65m")))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), triggerIDKey, triggerID))

			responseWriter := httptest.NewRecorder()
			updateTrigger(responseWriter, request)

			Convey("should return 200", func() {
				response := responseWriter.Result()
				defer response.Body.Close()
				So(response.StatusCode, ShouldEqual, http.StatusOK)
				So(isTriggerUpdated(response), ShouldBeTrue)
			})
		})

		Convey("with validate", func() {
			request := httptest.NewRequest("", fmt.Sprintf("/trigger?%s", validateFlag), bytes.NewBuffer(jsonTrigger))
			request.Header.Add("content-type", "application/json")
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "localMetricTTL", to.Duration("65m")))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), triggerIDKey, triggerID))

			responseWriter := httptest.NewRecorder()
			updateTrigger(responseWriter, request)

			Convey("should return 400 and tree of problems", func() {
				response := responseWriter.Result()
				defer response.Body.Close()

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

	Convey("Given is_remote flag is false and trigger_source is not set", t, func() {
		triggerId := "test"
		target := `test_target_value`

		jsonTrigger := MakeTestTriggerJson(target, triggerId, `"is_remote": false`)
		request := MakeTriggerData(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be graphite local", func() {
			SetupExpectations(localSource, db, target, triggerId, moira.GraphiteLocal)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	Convey("Given is_remote flag is true and trigger_source is not set", t, func() {
		triggerId := "test"
		target := `test_target_value`

		jsonTrigger := MakeTestTriggerJson(target, triggerId, `"is_remote": true`)
		request := MakeTriggerData(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be graphite remote", func() {
			SetupExpectations(remoteSource, db, target, triggerId, moira.GraphiteRemote)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	Convey("Given is_remote flag is not set and trigger_source is graphite_local", t, func() {
		triggerId := "test"
		target := `test_target_value`

		jsonTrigger := MakeTestTriggerJson(target, triggerId, `"trigger_source": "graphite_local"`)
		request := MakeTriggerData(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be graphite local", func() {
			SetupExpectations(localSource, db, target, triggerId, moira.GraphiteLocal)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	Convey("Given is_remote flag is not set and trigger_source is graphite_remote", t, func() {
		triggerId := "test"
		target := `test_target_value`

		jsonTrigger := MakeTestTriggerJson(target, triggerId, `"trigger_source": "graphite_remote"`)
		request := MakeTriggerData(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be graphite remote", func() {
			SetupExpectations(remoteSource, db, target, triggerId, moira.GraphiteRemote)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	Convey("Given is_remote flag is not set and trigger_source is prometheus_remote", t, func() {
		triggerId := "test"
		target := `test_target_value`

		jsonTrigger := MakeTestTriggerJson(target, triggerId, `"trigger_source": "prometheus_remote"`)
		request := MakeTriggerData(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be prometheus remote", func() {
			SetupExpectations(prometheusSource, db, target, triggerId, moira.PrometheusRemote)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	Convey("Given is_remote flag is true and trigger_source is graphite_local", t, func() {
		triggerId := "test"
		target := `test_target_value`

		jsonTrigger := MakeTestTriggerJson(target, triggerId, `"is_remote": true, "trigger_source": "graphite_local"`)
		request := MakeTriggerData(sourceProvider, triggerId, jsonTrigger)

		Convey("Expect trigger to be graphite local", func() {
			SetupExpectations(localSource, db, target, triggerId, moira.GraphiteLocal)

			responseWriter := httptest.NewRecorder()
			createTrigger(responseWriter, request)

			So(responseWriter.Code, ShouldEqual, 200)
		})
	})

	database = nil
}

func MakeTestTriggerJson(target, triggerId, triggerSource string) []byte {
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

func SetupExpectations(
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

func MakeTriggerData(
	sourceProvider *metricSource.SourceProvider,
	triggerId string,
	jsonTrigger []byte,
) *http.Request {
	const triggerIDKey = "triggerID"

	request := httptest.NewRequest("", "/trigger", bytes.NewBuffer(jsonTrigger))
	request.Header.Add("content-type", "application/json")
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "localMetricTTL", to.Duration("65m")))
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), triggerIDKey, triggerId))

	return request
}

func isTriggerUpdated(response *http.Response) bool {
	contentBytes, _ := io.ReadAll(response.Body)
	actual := dto.SaveTriggerResponse{}
	_ = json.Unmarshal(contentBytes, &actual)
	const expected = "trigger updated"

	return actual.Message == expected
}
