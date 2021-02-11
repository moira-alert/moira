package notifier

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

// Scheduler implements event scheduling functionality
type Scheduler interface {
	ScheduleNotification(now time.Time, event moira.NotificationEvent, trigger moira.TriggerData,
		contact moira.ContactData, plotting moira.PlottingData, throttledOld bool, sendFail int, logger moira.Logger) *moira.ScheduledNotification
}

// StandardScheduler represents standard event scheduling
type StandardScheduler struct {
	database moira.Database
	metrics  *metrics.NotifierMetrics
}

type throttlingLevel struct {
	duration time.Duration
	delay    time.Duration
	count    int64
}

// NewScheduler is initializer for StandardScheduler
func NewScheduler(database moira.Database, logger moira.Logger, metrics *metrics.NotifierMetrics) *StandardScheduler {
	return &StandardScheduler{
		database: database,
		metrics:  metrics,
	}
}

// ScheduleNotification is realization of scheduling event, based on trigger and subscription time intervals and triggers settings
func (scheduler *StandardScheduler) ScheduleNotification(now time.Time, event moira.NotificationEvent, trigger moira.TriggerData,
	contact moira.ContactData, plotting moira.PlottingData, throttledOld bool, sendFail int, logger moira.Logger) *moira.ScheduledNotification {

	var (
		next      time.Time
		throttled bool
	)
	if sendFail > 0 {
		next = now.Add(time.Minute)
		throttled = throttledOld
	} else {
		if event.State == moira.StateTEST {
			next = now
			throttled = false
		} else {
			next, throttled = scheduler.calculateNextDelivery(now, &event, logger)
		}
	}
	notification := &moira.ScheduledNotification{
		Event:     event,
		Trigger:   trigger,
		Contact:   contact,
		Throttled: throttled,
		SendFail:  sendFail,
		Timestamp: next.Unix(),
		Plotting:  plotting,
	}

	logger.Debugf("Scheduled notification at %s (%d)", next.Format("2006/01/02 15:04:05"), next.Unix())
	return notification
}

func (scheduler *StandardScheduler) calculateNextDelivery(now time.Time, event *moira.NotificationEvent,
	logger moira.Logger) (time.Time, bool) {
	// if trigger switches more than .count times in .length seconds, delay next delivery for .delay seconds
	// processing stops after first condition matches
	throttlingLevels := []throttlingLevel{
		{3 * time.Hour, time.Hour, 20},
		{time.Hour, time.Hour / 2, 10},
	}

	alarmFatigue := false

	next, beginning := scheduler.database.GetTriggerThrottling(event.TriggerID)

	if next.After(now) {
		alarmFatigue = true
	} else {
		next = now
	}

	subscription, err := scheduler.database.GetSubscription(moira.UseString(event.SubscriptionID))
	if err != nil {
		scheduler.metrics.SubsMalformed.Mark(1)
		logger.Debugf("Failed get subscription: %s", err.Error())
		return next, alarmFatigue
	}

	if subscription.ThrottlingEnabled {
		if next.After(now) {
			logger.Debugf("Using existing throttling, next at: %s", next)
		} else {
			for _, level := range throttlingLevels {
				from := now.Add(-level.duration)
				if from.Before(beginning) {
					from = beginning
				}
				count := scheduler.database.GetNotificationEventCount(event.TriggerID, from.Unix())
				if count >= level.count {
					next = now.Add(level.delay)
					logger.Debugf("Trigger switched %d times in last %s, delaying next notification for %s",
						count, level.duration, level.delay)
					if err = scheduler.database.SetTriggerThrottling(event.TriggerID, next); err != nil {
						logger.Errorf("Failed to set trigger throttling timestamp: %s", err)
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
		logger.Errorf("Failed to apply schedule: %s.", err)
	}
	return next, alarmFatigue
}

func calculateNextDelivery(schedule *moira.ScheduleData, nextTime time.Time) (time.Time, error) {
	if len(schedule.Days) != 0 && len(schedule.Days) != 7 {
		return nextTime, fmt.Errorf("invalid scheduled settings: %d days defined", len(schedule.Days))
	}

	if len(schedule.Days) == 0 {
		return nextTime, nil
	}
	beginOffset := time.Duration(schedule.StartOffset) * time.Minute
	endOffset := time.Duration(schedule.EndOffset) * time.Minute
	if schedule.EndOffset < schedule.StartOffset {
		endOffset = endOffset + (time.Hour * 24)
	}

	tzOffset := time.Duration(schedule.TimezoneOffset) * time.Minute
	localNextTime := nextTime.Add(-tzOffset).Truncate(time.Minute)
	localNextTimeDay := localNextTime.Truncate(24 * time.Hour) //nolint
	localNextWeekday := int(localNextTimeDay.Weekday()+6) % 7  //nolint

	if schedule.Days[localNextWeekday].Enabled &&
		(localNextTime.Equal(localNextTimeDay.Add(beginOffset)) || localNextTime.After(localNextTimeDay.Add(beginOffset))) &&
		(localNextTime.Equal(localNextTimeDay.Add(endOffset)) || localNextTime.Before(localNextTimeDay.Add(endOffset))) {
		return nextTime, nil
	}

	// find first allowed day
	for i := 0; i < 8; i++ {
		nextLocalDayBegin := localNextTimeDay.Add(time.Duration(i*24) * time.Hour) //nolint
		nextLocalWeekDay := int(nextLocalDayBegin.Weekday()+6) % 7                 //nolint
		if localNextTime.After(nextLocalDayBegin.Add(beginOffset)) {
			continue
		}
		if !schedule.Days[nextLocalWeekDay].Enabled {
			continue
		}
		return nextLocalDayBegin.Add(beginOffset + tzOffset), nil
	}

	return nextTime, fmt.Errorf("can not find allowed schedule day")
}
