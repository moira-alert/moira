package redis_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
)

func BenchmarkFetchNotificationsWithoutLimit(b *testing.B) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := redis.NewTestDatabase(logger)
	dataBase.Flush()

	for name, num := range TestCases {
		time.Sleep(time.Second)
		defer dataBase.Flush()
		name := name
		num := num

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
			}
		}

		addNotifications(dataBase, notifications) // nolint

		b.Run(name, func(b *testing.B) {
			dataBase.FetchNotificationsNoLimitOther(200) // nolint
		})
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
