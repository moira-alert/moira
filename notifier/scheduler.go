package notifier

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"time"
)

type Scheduler interface {
	ScheduleNotification(now time.Time, event moira_alert.EventData, trigger moira_alert.TriggerData, contact moira_alert.ContactData, throttledOld bool, sendfail int) *moira_alert.ScheduledNotification
}

type StandardScheduler struct {
	logger   moira_alert.Logger
	database moira_alert.Database
}

type throttlingLevel struct {
	duration time.Duration
	delay    time.Duration
	count    int64
}

func InitScheduler(database moira_alert.Database, logger moira_alert.Logger) *StandardScheduler {
	return &StandardScheduler{
		database: database,
		logger:   logger,
	}
}

func (scheduler *StandardScheduler) ScheduleNotification(now time.Time, event moira_alert.EventData, trigger moira_alert.TriggerData, contact moira_alert.ContactData, throttledOld bool, sendfail int) *moira_alert.ScheduledNotification {
	var (
		next      time.Time
		throttled bool
	)
	if sendfail > 0 {
		next = now.Add(time.Minute)
		throttled = throttledOld
	} else {
		if event.State == "TEST" {
			next = now
			throttled = false
		} else {
			next, throttled = scheduler.calculateNextDelivery(now, &event)
		}
	}
	notification := &moira_alert.ScheduledNotification{
		Event:     event,
		Trigger:   trigger,
		Contact:   contact,
		Throttled: throttled,
		SendFail:  sendfail,
		Timestamp: next.Unix(),
	}
	scheduler.logger.Debugf(
		"Scheduled notification for contact %s:%s trigger %s at %s (%d)",
		contact.Type, contact.Value, trigger.Name,
		next.Format("2006/01/02 15:04:05"), next.Unix())

	return notification
}

func (scheduler *StandardScheduler) calculateNextDelivery(now time.Time, event *moira_alert.EventData) (time.Time, bool) {
	// if trigger switches more than .count times in .length seconds, delay next delivery for .delay seconds
	// processing stops after first condition matches
	throttlingLevels := []throttlingLevel{
		{3 * time.Hour, time.Hour, 20},
		{time.Hour, time.Hour / 2, 10},
	}

	alarmFatigue := false

	next, beginning := scheduler.database.GetTriggerThrottlingTimestamps(event.TriggerID)

	if next.After(now) {
		alarmFatigue = true
	} else {
		next = now
	}

	subscription, err := scheduler.database.GetSubscription(event.SubscriptionID)
	if err != nil {
		scheduler.logger.Debugf("Failed get subscription by id: %s. %s", event.SubscriptionID, err.Error())
		return next, alarmFatigue
	}

	if subscription.ThrottlingEnabled {
		if next.After(now) {
			scheduler.logger.Debugf("Using existing throttling for trigger %s: %s", event.TriggerID, next)
		} else {
			for _, level := range throttlingLevels {
				from := now.Add(-level.duration)
				if from.Before(beginning) {
					from = beginning
				}
				count := scheduler.database.GetTriggerEventsCount(event.TriggerID, from.Unix())
				if count >= level.count {
					next = now.Add(level.delay)
					scheduler.logger.Debugf("Trigger %s switched %d times in last %s, delaying next notification for %s", event.TriggerID, count, level.duration, level.delay)
					if err := scheduler.database.SetTriggerThrottlingTimestamp(event.TriggerID, next); err != nil {
						scheduler.logger.Errorf("Failed to set trigger throttling timestamp: %s", err)
					}
					alarmFatigue = true
					break
				} else if count == level.count-1 {
					alarmFatigue = true
				}
			}
		}
	} else {
		next = now
	}
	next, err = calculateNextDelivery(&subscription.Schedule, next)
	if err != nil {
		scheduler.logger.Errorf("Failed to aply schedule for subscriptionID: %s. %s.", event.SubscriptionID, err)
	}
	return next, alarmFatigue
}

func calculateNextDelivery(schedule *moira_alert.ScheduleData, nextTime time.Time) (time.Time, error) {

	if len(schedule.Days) != 0 && len(schedule.Days) != 7 {
		return nextTime, fmt.Errorf("Invalid scheduled settings: %d days defined", len(schedule.Days))
	}

	if len(schedule.Days) == 0 {
		return nextTime, nil
	}

	tzOffset := time.Duration(schedule.TimezoneOffset) * time.Minute
	localNextTime := nextTime.Add(-tzOffset).Truncate(time.Minute)
	beginOffset := time.Duration(schedule.StartOffset) * time.Minute
	endOffset := time.Duration(schedule.EndOffset) * time.Minute
	localNextTimeDay := localNextTime.Truncate(24 * time.Hour)
	localNextWeekday := int(localNextTimeDay.Weekday()+6) % 7

	if schedule.Days[localNextWeekday].Enabled &&
		(localNextTime.Equal(localNextTimeDay.Add(beginOffset)) || localNextTime.After(localNextTimeDay.Add(beginOffset))) &&
		(localNextTime.Equal(localNextTimeDay.Add(endOffset)) || localNextTime.Before(localNextTimeDay.Add(endOffset))) {
		return nextTime, nil
	}

	// find first allowed day
	for i := 0; i < 8; i++ {
		nextLocalDayBegin := localNextTimeDay.Add(time.Duration(i*24) * time.Hour)
		nextLocalWeekDay := int(nextLocalDayBegin.Weekday()+6) % 7
		if localNextTime.After(nextLocalDayBegin.Add(beginOffset)) {
			continue
		}
		if !schedule.Days[nextLocalWeekDay].Enabled {
			continue
		}
		return nextLocalDayBegin.Add(beginOffset + tzOffset), nil
	}

	return nextTime, fmt.Errorf("Can not find allowed schedule day")
}
