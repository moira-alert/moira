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

func TestFetchNotificationsForTest(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := redis.NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	now := time.Now().Unix()
	notificationNew := moira.ScheduledNotification{
		SendFail:  1,
		Timestamp: now + dataBase.GetDelayedTimeInSeconds(),
		CreatedAt: now,
	}
	notification := moira.ScheduledNotification{
		SendFail:  2,
		Timestamp: now,
		CreatedAt: now,
	}
	notificationOld := moira.ScheduledNotification{
		SendFail:  3,
		Timestamp: now - dataBase.GetDelayedTimeInSeconds(),
		CreatedAt: now,
	}

	addNotifications(dataBase, []moira.ScheduledNotification{notificationOld, notification, notificationNew})
	wg := sync.WaitGroup{}
	var limit int64 = 2
	wg.Add(3)
	go func() {
		_, err := dataBase.FetchNotificationsForTest(&wg, now+dataBase.GetDelayedTimeInSeconds()*2, limit, "notifier-1") //nolint
		if err != nil {
			log.Printf("notifier-1: %v\n", err)
		}
	}()
	go func() {
		_, err := dataBase.FetchNotificationsForTest(&wg, now+dataBase.GetDelayedTimeInSeconds()*2, limit, "notifier-2") //nolint
		if err != nil {
			log.Printf("notifier-2: %v\n", err)
		}
	}()
	// time.Sleep(100 * time.Millisecond)
	go func() {
		_, err := dataBase.FetchNotificationsForTest(&wg, now+dataBase.GetDelayedTimeInSeconds()*2, limit, "notifier-3") //nolint
		if err != nil {
			log.Printf("notifier-3: %v\n", err)
		}
	}()
	wg.Wait()

	assert.True(t, false)
}
