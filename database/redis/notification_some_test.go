package redis_test

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
)

var TestCases map[string]int = map[string]int{
	"With 100 notifications":   100,
	"With 1000 notifications":  1000,
	"With 10000 notifications": 10000,
	// "With 50000 notifications": 50000,
}

func addNotifications(dataBase *redis.DbConnector, notifications []moira.ScheduledNotification) {
	for _, notification := range notifications {
		dataBase.AddNotification(&notification) // nolint
	}
}

func getDelayedNotification(n int) moira.ScheduledNotification {
	return moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: fmt.Sprintf("test%v", n),
		},
		Timestamp: 2000,
		CreatedAt: 100,
	}
}

func getNotDelayedNotification() moira.ScheduledNotification {
	return moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test",
		},
		Timestamp: 120,
		CreatedAt: 100,
	}
}

func sum(arr []float64) float64 {
	var s float64
	for i := range arr {
		s += arr[i]
	}
	return s
}

func TestFetchNotificationsWithoutLimit(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := redis.NewTestDatabase(logger)
	dataBase.Flush()

	for name, num := range TestCases {
		defer dataBase.Flush()
		name := name
		num := num

		times := make([]float64, 10)
		for z := range times {
			step := num / 10
			for i := 0; i < step/2; i++ {
				_ = dataBase.SetTriggerLastCheck(fmt.Sprintf("test%v", i), &moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test": {},
					},
					Maintenance:                  140,
					LastSuccessfulCheckTimestamp: 130,
				}, moira.TriggerSourceNotSet) // nolint
			}

			notifications := make([]moira.ScheduledNotification, 0, num)
			notifications = append(notifications, moira.ScheduledNotification{
				Trigger: moira.TriggerData{
					ID: "test",
				},
				Timestamp: 120,
				CreatedAt: 100,
			})

			for i := 0; i < num; i++ {
				if i%step == 0 {
					for j := 0; j < step/2; j++ {
						notifications = append(notifications, getDelayedNotification(j))
					}

					for j := step / 2; j < step; j++ {
						notifications = append(notifications, getNotDelayedNotification())
					}

					// for j := 0; j < step; j++ {
					// 	notifications = append(notifications, getNotDelayedNotification())
					// }
				}
			}

			addNotifications(dataBase, notifications) // nolint

			start := time.Now()
			dataBase.FetchNotificationsNoLimitOther(3000) // nolint
			end := time.Now()
			times[z] = end.Sub(start).Seconds()
		}

		log.Printf("%s: average %v in seconds", name, sum(times)/float64(len(times)))
	}
}

// func BenchmarkFetchNotificationsWithoutLimitOther(b *testing.B) {
// 	logger, _ := logging.GetLogger("dataBase")
// 	dataBase := redis.NewTestDatabase(logger)
// 	dataBase.Flush()

// 	for name, num := range TestCases {
// 		time.Sleep(time.Second)
// 		defer dataBase.Flush()
// 		name := name
// 		num := num

// 		step := num / 10
// 		for i := 0; i < step/2; i++ {
// 			dataBase.SetTriggerLastCheck(fmt.Sprintf("test%v", i), &moira.CheckData{
// 				Metrics: map[string]moira.MetricState{
// 					"test": {},
// 				},
// 				Maintenance:                  140,
// 				LastSuccessfulCheckTimestamp: 130,
// 			}, moira.TriggerSourceNotSet)
// 		}

// 		notifications := make([]moira.ScheduledNotification, 0, num)
// 		notifications = append(notifications, moira.ScheduledNotification{
// 			Trigger: moira.TriggerData{
// 				ID: "test",
// 			},
// 			Timestamp: 120,
// 			CreatedAt: 100,
// 		})

// 		for i := 0; i < num; i++ {
// 			if i%step == 0 {
// 				for j := 0; j < step/2; j++ {
// 					notifications = append(notifications, getDelayedNotification(j))
// 				}

// 				for j := step / 2; j < step; j++ {
// 					notifications = append(notifications, getNotDelayedNotification())
// 				}
// 			}
// 		}

// 		addNotifications(dataBase, notifications)

// 		b.Run(name+"_other", func(b *testing.B) {
// 			dataBase.FetchNotificationsNoLimitOther(200)
// 		})
// 	}
// }

// func BenchmarkFetchNotificationsWithLimit(b *testing.B) {
// 	logger, _ := logging.GetLogger("dataBase")
// 	dataBase := redis.NewTestDatabase(logger)
// 	dataBase.Flush()

// 	for name, num := range TestCases {
// 		time.Sleep(time.Second)
// 		defer dataBase.Flush()
// 		name := name
// 		num := num

// 		step := num / 10
// 		for i := 0; i < step/2; i++ {
// 			dataBase.SetTriggerLastCheck(fmt.Sprintf("test%v", i), &moira.CheckData{
// 				Metrics: map[string]moira.MetricState{
// 					"test": {},
// 				},
// 				Maintenance:                  140,
// 				LastSuccessfulCheckTimestamp: 130,
// 			}, moira.TriggerSourceNotSet)
// 		}

// 		notifications := make([]moira.ScheduledNotification, 0, num)
// 		notifications = append(notifications, moira.ScheduledNotification{
// 			Trigger: moira.TriggerData{
// 				ID: "test",
// 			},
// 			Timestamp: 120,
// 			CreatedAt: 100,
// 		})

// 		for i := 0; i < num; i++ {
// 			if i%step == 0 {
// 				for j := 0; j < step/2; j++ {
// 					notifications = append(notifications, getDelayedNotification(j))
// 				}

// 				for j := step / 2; j < step; j++ {
// 					notifications = append(notifications, getNotDelayedNotification())
// 				}
// 			}
// 		}

// 		addNotifications(dataBase, notifications)

// 		b.Run(name, func(b *testing.B) {
// 			dataBase.FetchNotificationsWithLimitDo(200, 50)
// 		})
// 	}
// }

// func BenchmarkFetchNotificationsWithLimitOther(b *testing.B) {
// 	logger, _ := logging.GetLogger("dataBase")
// 	dataBase := redis.NewTestDatabase(logger)
// 	dataBase.Flush()

// 	for name, num := range TestCases {
// 		time.Sleep(time.Second)
// 		defer dataBase.Flush()
// 		name := name
// 		num := num

// 		step := num / 10
// 		for i := 0; i < step/2; i++ {
// 			dataBase.SetTriggerLastCheck(fmt.Sprintf("test%v", i), &moira.CheckData{
// 				Metrics: map[string]moira.MetricState{
// 					"test": {},
// 				},
// 				Maintenance:                  140,
// 				LastSuccessfulCheckTimestamp: 130,
// 			}, moira.TriggerSourceNotSet)
// 		}

// 		notifications := make([]moira.ScheduledNotification, 0, num)
// 		notifications = append(notifications, moira.ScheduledNotification{
// 			Trigger: moira.TriggerData{
// 				ID: "test",
// 			},
// 			Timestamp: 120,
// 			CreatedAt: 100,
// 		})

// 		for i := 0; i < num; i++ {
// 			if i%step == 0 {
// 				for j := 0; j < step/2; j++ {
// 					notifications = append(notifications, getDelayedNotification(j))
// 				}

// 				for j := step / 2; j < step; j++ {
// 					notifications = append(notifications, getNotDelayedNotification())
// 				}
// 			}
// 		}

// 		addNotifications(dataBase, notifications)

// 		b.Run(name+"_other", func(b *testing.B) {
// 			dataBase.FetchNotificationsWithLimitDoOther(200, 50)
// 		})
// 	}
// }
