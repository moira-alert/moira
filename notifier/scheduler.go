package notifier

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

// Scheduler implements event scheduling functionality.
type Scheduler interface {
	ScheduleNotification(params moira.SchedulerParams, logger moira.Logger) *moira.ScheduledNotification
}

// SchedulerConfig is a list of immutable params for Scheduler
type SchedulerConfig struct {
	ReschedulingDelay time.Duration
}

// StandardScheduler represents standard event scheduling.
type StandardScheduler struct {
	database moira.Database
	metrics  *metrics.NotifierMetrics
	config   SchedulerConfig
}

type throttlingLevel struct {
	duration time.Duration
	delay    time.Duration
	count    int64
}

// NewScheduler is initializer for StandardScheduler.
func NewScheduler(database moira.Database, logger moira.Logger, metrics *metrics.NotifierMetrics, config SchedulerConfig,
) *StandardScheduler {
	return &StandardScheduler{
		database: database,
		metrics:  metrics,
		config:   config,
	}
}

// ScheduleNotification is realization of scheduling event, based on trigger and subscription time intervals and triggers settings.
func (scheduler *StandardScheduler) ScheduleNotification(params moira.SchedulerParams, logger moira.Logger,
) *moira.ScheduledNotification {
	var (
		next      time.Time
		throttled bool
	)
	if params.SendFail > 0 {
		next = params.Now.Add(scheduler.config.ReschedulingDelay)
		throttled = params.ThrottledOld
	} else {
		if params.Event.State == moira.StateTEST {
			next = params.Now
			throttled = false
		} else {
			next, throttled = scheduler.calculateNextDelivery(params.Now, &params.Event, logger)
		}
	}
	notification := &moira.ScheduledNotification{
		Event:     params.Event,
		Trigger:   params.Trigger,
		Contact:   params.Contact,
		Throttled: throttled,
		SendFail:  params.SendFail,
		Timestamp: next.Unix(),
		CreatedAt: params.Now.Unix(),
		Plotting:  params.Plotting,
	}

	logger.Debug().
		String("notification_timestamp", next.Format("2006/01/02 15:04:05")).
		Int64("notification_timestamp_unix", next.Unix()).
		Int64("notification_created_at_unix", params.Now.Unix()).
		Msg("Scheduled notification")
	return notification
}

func (scheduler *StandardScheduler) calculateNextDelivery(now time.Time, event *moira.NotificationEvent,
	logger moira.Logger,
) (time.Time, bool) {
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
		logger.Debug().
			Error(err).
			Msg("Failed get subscription")
		return next, alarmFatigue
	}

	if subscription.ThrottlingEnabled {
		if next.After(now) {
			logger.Debug().
				String("next_at", next.String()).
				Msg("Using existing throttling")
		} else {
			for _, level := range throttlingLevels {
				from := now.Add(-level.duration)
				if from.Before(beginning) {
					from = beginning
				}
				count := scheduler.database.GetNotificationEventCount(event.TriggerID, from.Unix())
				if count >= level.count {
					next = now.Add(level.delay)
					logger.Debug().
						Int64("trigger_switched_times", count).
						String("in_duration", level.duration.String()).
						String("delaying_for", level.delay.String()).
						Msg("Trigger switched many times, delaying next notification for some time")

					if err = scheduler.database.SetTriggerThrottling(event.TriggerID, next); err != nil {
						logger.Error().
							Error(err).
							Msg("Failed to set trigger throttling timestamp")
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
		logger.Error().
			Error(err).
			Msg("Failed to apply schedule")
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
		endOffset += time.Hour * 24
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
