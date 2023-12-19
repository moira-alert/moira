package redis_test

import (
	"log"
	"sync"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/stretchr/testify/assert"
)

func addNotifications(database *redis.DbConnector, notifications []moira.ScheduledNotification) {
	for _, notification := range notifications {
		err := database.AddNotification(&notification)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func TestFetchNotificationsForTest(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := redis.NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	now := time.Now().Unix()
	delayedTime := int64((60 * time.Second).Seconds())
	notificationNew := moira.ScheduledNotification{
		SendFail:  1,
		Timestamp: now + delayedTime,
		CreatedAt: now,
	}
	notification := moira.ScheduledNotification{
		SendFail:  2,
		Timestamp: now,
		CreatedAt: now,
	}
	notificationOld := moira.ScheduledNotification{
		SendFail:  3,
		Timestamp: now - delayedTime,
		CreatedAt: now - delayedTime,
	}

	addNotifications(dataBase, []moira.ScheduledNotification{notificationOld, notification, notificationNew})
	wg := sync.WaitGroup{}
	var limit int64 = 2
	wg.Add(1)
	go func() {
		// _, err := dataBase.FetchNotificationsForTest(&wg, now+delayedTime*2, limit, "notifier-1") //nolint
		// if err != nil {
		// 	log.Printf("notifier-1: %v\n", err)
		// }
		defer wg.Done()
		if _, err := dataBase.FetchNotifications(now+delayedTime*2, limit); err != nil {
			log.Printf("notifier-1: %v\n", err)
		}
	}()
	// go func() {
	// 	// _, err := dataBase.FetchNotificationsForTest(&wg, now+delayedTime*2, limit, "notifier-2") //nolint
	// 	// if err != nil {
	// 	// 	log.Printf("notifier-2: %v\n", err)
	// 	// }
	// 	defer wg.Done()
	// 	if _, err := dataBase.FetchNotifications(now+delayedTime*2, limit); err != nil {
	// 		log.Printf("notifier-2: %v\n", err)
	// 	}
	// }()
	// // time.Sleep(100 * time.Millisecond)
	// go func() {
	// 	// _, err := dataBase.FetchNotificationsForTest(&wg, now+delayedTime*2, limit, "notifier-3") //nolint
	// 	// if err != nil {
	// 	// 	log.Printf("notifier-3: %v\n", err)
	// 	// }
	// 	defer wg.Done()
	// 	if _, err := dataBase.FetchNotifications(now+delayedTime*2, limit); err != nil {
	// 		log.Printf("notifier-3: %v\n", err)
	// 	}
	// }()
	wg.Wait()

	notifications, count, err := dataBase.GetNotifications(0, -1)
	if err != nil {
		log.Println("error: ", err)
	}

	log.Println("count: ", count)
	for _, notification := range notifications {
		log.Println(notification)
	}

	assert.True(t, false)
}
