package database

// DO NOT EDIT!
// This code is generated with http://github.com/hexdigest/gowrap tool
// using ../metrics/gen_templates/moira-metrics template

//go:generate gowrap gen -p github.com/moira-alert/moira -i Database -t ../metrics/gen_templates/moira-metrics -o database_with_metrics.go

import (
	"sync"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
	"gopkg.in/tomb.v2"
)

// DatabaseWithMetrics implements moira.Database interface with all methods wrapped by moira metrics
type DatabaseWithMetrics struct {
	base          moira.Database
	metricsPrefix string
	registry      *metrics.Registry
	timers        *[84]*metrics.Timer
	mutex         sync.Mutex
}

// NewDatabaseWithMetrics returns an instance of the moira.Database decorated with duration metric
func NewDatabaseWithMetrics(base moira.Database, metricsPrefix string, registry *metrics.Registry) *DatabaseWithMetrics {
	return &DatabaseWithMetrics{
		base:          base,
		metricsPrefix: metricsPrefix,
		registry:      registry,
		timers:        &[84]*metrics.Timer{},
	}
}

// GetLocalTriggersToCheckCount implements moira.Database
func (d *DatabaseWithMetrics) GetLocalTriggersToCheckCount() (i1 int64, err error) {
	since := time.Now()
	if d.timers[0] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[0] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetLocalTriggersToCheckCount", "duration")
			d.timers[0] = &t
		}
	}
	defer (*d.timers[0]).UpdateSince(since)
	return d.base.GetLocalTriggersToCheckCount()
}

// UpdateMetricsHeartbeat implements moira.Database
func (d *DatabaseWithMetrics) UpdateMetricsHeartbeat() (err error) {
	since := time.Now()
	if d.timers[1] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[1] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "UpdateMetricsHeartbeat", "duration")
			d.timers[1] = &t
		}
	}
	defer (*d.timers[1]).UpdateSince(since)
	return d.base.UpdateMetricsHeartbeat()
}

// GetNotifierState implements moira.Database
func (d *DatabaseWithMetrics) GetNotifierState() (s1 string, err error) {
	since := time.Now()
	if d.timers[2] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[2] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetNotifierState", "duration")
			d.timers[2] = &t
		}
	}
	defer (*d.timers[2]).UpdateSince(since)
	return d.base.GetNotifierState()
}

// SetTriggerLastCheck implements moira.Database
func (d *DatabaseWithMetrics) SetTriggerLastCheck(triggerID string, checkData *moira.CheckData, isRemote bool) (err error) {
	since := time.Now()
	if d.timers[3] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[3] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "SetTriggerLastCheck", "duration")
			d.timers[3] = &t
		}
	}
	defer (*d.timers[3]).UpdateSince(since)
	return d.base.SetTriggerLastCheck(triggerID, checkData, isRemote)
}

// GetNotificationEvents implements moira.Database
func (d *DatabaseWithMetrics) GetNotificationEvents(triggerID string, start int64, size int64) (npa1 []*moira.NotificationEvent, err error) {
	since := time.Now()
	if d.timers[4] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[4] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetNotificationEvents", "duration")
			d.timers[4] = &t
		}
	}
	defer (*d.timers[4]).UpdateSince(since)
	return d.base.GetNotificationEvents(triggerID, start, size)
}

// GetPatternMetrics implements moira.Database
func (d *DatabaseWithMetrics) GetPatternMetrics(pattern string) (sa1 []string, err error) {
	since := time.Now()
	if d.timers[5] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[5] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetPatternMetrics", "duration")
			d.timers[5] = &t
		}
	}
	defer (*d.timers[5]).UpdateSince(since)
	return d.base.GetPatternMetrics(pattern)
}

// SaveMetrics implements moira.Database
func (d *DatabaseWithMetrics) SaveMetrics(buffer map[string]*moira.MatchedMetric) (err error) {
	since := time.Now()
	if d.timers[6] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[6] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "SaveMetrics", "duration")
			d.timers[6] = &t
		}
	}
	defer (*d.timers[6]).UpdateSince(since)
	return d.base.SaveMetrics(buffer)
}

// SetNotifierState implements moira.Database
func (d *DatabaseWithMetrics) SetNotifierState(s1 string) (err error) {
	since := time.Now()
	if d.timers[7] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[7] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "SetNotifierState", "duration")
			d.timers[7] = &t
		}
	}
	defer (*d.timers[7]).UpdateSince(since)
	return d.base.SetNotifierState(s1)
}

// GetRemoteTriggerIDs implements moira.Database
func (d *DatabaseWithMetrics) GetRemoteTriggerIDs() (sa1 []string, err error) {
	since := time.Now()
	if d.timers[8] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[8] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetRemoteTriggerIDs", "duration")
			d.timers[8] = &t
		}
	}
	defer (*d.timers[8]).UpdateSince(since)
	return d.base.GetRemoteTriggerIDs()
}

// GetTriggerThrottling implements moira.Database
func (d *DatabaseWithMetrics) GetTriggerThrottling(triggerID string) (t1 time.Time, t2 time.Time) {
	since := time.Now()
	if d.timers[9] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[9] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetTriggerThrottling", "duration")
			d.timers[9] = &t
		}
	}
	defer (*d.timers[9]).UpdateSince(since)
	return d.base.GetTriggerThrottling(triggerID)
}

// FetchNotifications implements moira.Database
func (d *DatabaseWithMetrics) FetchNotifications(to int64, limit int64) (spa1 []*moira.ScheduledNotification, err error) {
	since := time.Now()
	if d.timers[10] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[10] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "FetchNotifications", "duration")
			d.timers[10] = &t
		}
	}
	defer (*d.timers[10]).UpdateSince(since)
	return d.base.FetchNotifications(to, limit)
}

// SetUsernameID implements moira.Database
func (d *DatabaseWithMetrics) SetUsernameID(messenger string, username string, id string) (err error) {
	since := time.Now()
	if d.timers[11] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[11] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "SetUsernameID", "duration")
			d.timers[11] = &t
		}
	}
	defer (*d.timers[11]).UpdateSince(since)
	return d.base.SetUsernameID(messenger, username, id)
}

// RemoveTrigger implements moira.Database
func (d *DatabaseWithMetrics) RemoveTrigger(triggerID string) (err error) {
	since := time.Now()
	if d.timers[12] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[12] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemoveTrigger", "duration")
			d.timers[12] = &t
		}
	}
	defer (*d.timers[12]).UpdateSince(since)
	return d.base.RemoveTrigger(triggerID)
}

// GetMetricsTTLSeconds implements moira.Database
func (d *DatabaseWithMetrics) GetMetricsTTLSeconds() (i1 int64) {
	since := time.Now()
	if d.timers[13] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[13] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetMetricsTTLSeconds", "duration")
			d.timers[13] = &t
		}
	}
	defer (*d.timers[13]).UpdateSince(since)
	return d.base.GetMetricsTTLSeconds()
}

// AddRemoteTriggersToCheck implements moira.Database
func (d *DatabaseWithMetrics) AddRemoteTriggersToCheck(triggerIDs []string) (err error) {
	since := time.Now()
	if d.timers[14] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[14] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "AddRemoteTriggersToCheck", "duration")
			d.timers[14] = &t
		}
	}
	defer (*d.timers[14]).UpdateSince(since)
	return d.base.AddRemoteTriggersToCheck(triggerIDs)
}

// GetRemoteTriggersToCheckCount implements moira.Database
func (d *DatabaseWithMetrics) GetRemoteTriggersToCheckCount() (i1 int64, err error) {
	since := time.Now()
	if d.timers[15] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[15] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetRemoteTriggersToCheckCount", "duration")
			d.timers[15] = &t
		}
	}
	defer (*d.timers[15]).UpdateSince(since)
	return d.base.GetRemoteTriggersToCheckCount()
}

// SetTriggerCheckMaintenance implements moira.Database
func (d *DatabaseWithMetrics) SetTriggerCheckMaintenance(triggerID string, metrics map[string]int64, triggerMaintenance *int64, userLogin string, timeCallMaintenance int64) (err error) {
	since := time.Now()
	if d.timers[16] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[16] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "SetTriggerCheckMaintenance", "duration")
			d.timers[16] = &t
		}
	}
	defer (*d.timers[16]).UpdateSince(since)
	return d.base.SetTriggerCheckMaintenance(triggerID, metrics, triggerMaintenance, userLogin, timeCallMaintenance)
}

// GetNotificationEventCount implements moira.Database
func (d *DatabaseWithMetrics) GetNotificationEventCount(triggerID string, from int64) (i1 int64) {
	since := time.Now()
	if d.timers[17] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[17] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetNotificationEventCount", "duration")
			d.timers[17] = &t
		}
	}
	defer (*d.timers[17]).UpdateSince(since)
	return d.base.GetNotificationEventCount(triggerID, from)
}

// MarkTriggersAsUnused implements moira.Database
func (d *DatabaseWithMetrics) MarkTriggersAsUnused(triggerIDs ...string) (err error) {
	since := time.Now()
	if d.timers[18] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[18] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "MarkTriggersAsUnused", "duration")
			d.timers[18] = &t
		}
	}
	defer (*d.timers[18]).UpdateSince(since)
	return d.base.MarkTriggersAsUnused(triggerIDs...)
}

// FetchTriggersToReindex implements moira.Database
func (d *DatabaseWithMetrics) FetchTriggersToReindex(from int64) (sa1 []string, err error) {
	since := time.Now()
	if d.timers[19] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[19] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "FetchTriggersToReindex", "duration")
			d.timers[19] = &t
		}
	}
	defer (*d.timers[19]).UpdateSince(since)
	return d.base.FetchTriggersToReindex(from)
}

// MarkTriggersAsUsed implements moira.Database
func (d *DatabaseWithMetrics) MarkTriggersAsUsed(triggerIDs ...string) (err error) {
	since := time.Now()
	if d.timers[20] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[20] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "MarkTriggersAsUsed", "duration")
			d.timers[20] = &t
		}
	}
	defer (*d.timers[20]).UpdateSince(since)
	return d.base.MarkTriggersAsUsed(triggerIDs...)
}

// GetTrigger implements moira.Database
func (d *DatabaseWithMetrics) GetTrigger(triggerID string) (t1 moira.Trigger, err error) {
	since := time.Now()
	if d.timers[21] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[21] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetTrigger", "duration")
			d.timers[21] = &t
		}
	}
	defer (*d.timers[21]).UpdateSince(since)
	return d.base.GetTrigger(triggerID)
}

// GetTriggerChecks implements moira.Database
func (d *DatabaseWithMetrics) GetTriggerChecks(triggerIDs []string) (tpa1 []*moira.TriggerCheck, err error) {
	since := time.Now()
	if d.timers[22] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[22] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetTriggerChecks", "duration")
			d.timers[22] = &t
		}
	}
	defer (*d.timers[22]).UpdateSince(since)
	return d.base.GetTriggerChecks(triggerIDs)
}

// SaveTrigger implements moira.Database
func (d *DatabaseWithMetrics) SaveTrigger(triggerID string, trigger *moira.Trigger) (err error) {
	since := time.Now()
	if d.timers[23] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[23] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "SaveTrigger", "duration")
			d.timers[23] = &t
		}
	}
	defer (*d.timers[23]).UpdateSince(since)
	return d.base.SaveTrigger(triggerID, trigger)
}

// RemoveAllNotificationEvents implements moira.Database
func (d *DatabaseWithMetrics) RemoveAllNotificationEvents() (err error) {
	since := time.Now()
	if d.timers[24] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[24] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemoveAllNotificationEvents", "duration")
			d.timers[24] = &t
		}
	}
	defer (*d.timers[24]).UpdateSince(since)
	return d.base.RemoveAllNotificationEvents()
}

// AddPatternMetric implements moira.Database
func (d *DatabaseWithMetrics) AddPatternMetric(pattern string, metric string) (err error) {
	since := time.Now()
	if d.timers[25] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[25] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "AddPatternMetric", "duration")
			d.timers[25] = &t
		}
	}
	defer (*d.timers[25]).UpdateSince(since)
	return d.base.AddPatternMetric(pattern, metric)
}

// RemovePattern implements moira.Database
func (d *DatabaseWithMetrics) RemovePattern(pattern string) (err error) {
	since := time.Now()
	if d.timers[26] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[26] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemovePattern", "duration")
			d.timers[26] = &t
		}
	}
	defer (*d.timers[26]).UpdateSince(since)
	return d.base.RemovePattern(pattern)
}

// AllowStale implements moira.Database
func (d *DatabaseWithMetrics) AllowStale() (d1 moira.Database) {
	since := time.Now()
	if d.timers[27] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[27] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "AllowStale", "duration")
			d.timers[27] = &t
		}
	}
	defer (*d.timers[27]).UpdateSince(since)
	return d.base.AllowStale()
}

// DeleteTriggerThrottling implements moira.Database
func (d *DatabaseWithMetrics) DeleteTriggerThrottling(triggerID string) (err error) {
	since := time.Now()
	if d.timers[28] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[28] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "DeleteTriggerThrottling", "duration")
			d.timers[28] = &t
		}
	}
	defer (*d.timers[28]).UpdateSince(since)
	return d.base.DeleteTriggerThrottling(triggerID)
}

// SaveSubscription implements moira.Database
func (d *DatabaseWithMetrics) SaveSubscription(subscription *moira.SubscriptionData) (err error) {
	since := time.Now()
	if d.timers[29] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[29] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "SaveSubscription", "duration")
			d.timers[29] = &t
		}
	}
	defer (*d.timers[29]).UpdateSince(since)
	return d.base.SaveSubscription(subscription)
}

// GetUserSubscriptionIDs implements moira.Database
func (d *DatabaseWithMetrics) GetUserSubscriptionIDs(userLogin string) (sa1 []string, err error) {
	since := time.Now()
	if d.timers[30] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[30] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetUserSubscriptionIDs", "duration")
			d.timers[30] = &t
		}
	}
	defer (*d.timers[30]).UpdateSince(since)
	return d.base.GetUserSubscriptionIDs(userLogin)
}

// AddNotification implements moira.Database
func (d *DatabaseWithMetrics) AddNotification(notification *moira.ScheduledNotification) (err error) {
	since := time.Now()
	if d.timers[31] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[31] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "AddNotification", "duration")
			d.timers[31] = &t
		}
	}
	defer (*d.timers[31]).UpdateSince(since)
	return d.base.AddNotification(notification)
}

// DeleteTriggerCheckLock implements moira.Database
func (d *DatabaseWithMetrics) DeleteTriggerCheckLock(triggerID string) (err error) {
	since := time.Now()
	if d.timers[32] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[32] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "DeleteTriggerCheckLock", "duration")
			d.timers[32] = &t
		}
	}
	defer (*d.timers[32]).UpdateSince(since)
	return d.base.DeleteTriggerCheckLock(triggerID)
}

// GetContact implements moira.Database
func (d *DatabaseWithMetrics) GetContact(contactID string) (c1 moira.ContactData, err error) {
	since := time.Now()
	if d.timers[33] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[33] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetContact", "duration")
			d.timers[33] = &t
		}
	}
	defer (*d.timers[33]).UpdateSince(since)
	return d.base.GetContact(contactID)
}

// SaveSubscriptions implements moira.Database
func (d *DatabaseWithMetrics) SaveSubscriptions(subscriptions []*moira.SubscriptionData) (err error) {
	since := time.Now()
	if d.timers[34] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[34] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "SaveSubscriptions", "duration")
			d.timers[34] = &t
		}
	}
	defer (*d.timers[34]).UpdateSince(since)
	return d.base.SaveSubscriptions(subscriptions)
}

// GetLocalTriggerIDs implements moira.Database
func (d *DatabaseWithMetrics) GetLocalTriggerIDs() (sa1 []string, err error) {
	since := time.Now()
	if d.timers[35] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[35] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetLocalTriggerIDs", "duration")
			d.timers[35] = &t
		}
	}
	defer (*d.timers[35]).UpdateSince(since)
	return d.base.GetLocalTriggerIDs()
}

// SaveContact implements moira.Database
func (d *DatabaseWithMetrics) SaveContact(contact *moira.ContactData) (err error) {
	since := time.Now()
	if d.timers[36] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[36] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "SaveContact", "duration")
			d.timers[36] = &t
		}
	}
	defer (*d.timers[36]).UpdateSince(since)
	return d.base.SaveContact(contact)
}

// RemoveMetricValues implements moira.Database
func (d *DatabaseWithMetrics) RemoveMetricValues(metric string, toTime int64) (err error) {
	since := time.Now()
	if d.timers[37] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[37] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemoveMetricValues", "duration")
			d.timers[37] = &t
		}
	}
	defer (*d.timers[37]).UpdateSince(since)
	return d.base.RemoveMetricValues(metric, toTime)
}

// GetIDByUsername implements moira.Database
func (d *DatabaseWithMetrics) GetIDByUsername(messenger string, username string) (s1 string, err error) {
	since := time.Now()
	if d.timers[38] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[38] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetIDByUsername", "duration")
			d.timers[38] = &t
		}
	}
	defer (*d.timers[38]).UpdateSince(since)
	return d.base.GetIDByUsername(messenger, username)
}

// GetTagNames implements moira.Database
func (d *DatabaseWithMetrics) GetTagNames() (sa1 []string, err error) {
	since := time.Now()
	if d.timers[39] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[39] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetTagNames", "duration")
			d.timers[39] = &t
		}
	}
	defer (*d.timers[39]).UpdateSince(since)
	return d.base.GetTagNames()
}

// SaveTriggersSearchResults implements moira.Database
func (d *DatabaseWithMetrics) SaveTriggersSearchResults(searchResultsID string, searchResults []*moira.SearchResult) (err error) {
	since := time.Now()
	if d.timers[40] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[40] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "SaveTriggersSearchResults", "duration")
			d.timers[40] = &t
		}
	}
	defer (*d.timers[40]).UpdateSince(since)
	return d.base.SaveTriggersSearchResults(searchResultsID, searchResults)
}

// RemoveSubscription implements moira.Database
func (d *DatabaseWithMetrics) RemoveSubscription(subscriptionID string) (err error) {
	since := time.Now()
	if d.timers[41] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[41] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemoveSubscription", "duration")
			d.timers[41] = &t
		}
	}
	defer (*d.timers[41]).UpdateSince(since)
	return d.base.RemoveSubscription(subscriptionID)
}

// GetTagsSubscriptions implements moira.Database
func (d *DatabaseWithMetrics) GetTagsSubscriptions(tags []string) (spa1 []*moira.SubscriptionData, err error) {
	since := time.Now()
	if d.timers[42] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[42] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetTagsSubscriptions", "duration")
			d.timers[42] = &t
		}
	}
	defer (*d.timers[42]).UpdateSince(since)
	return d.base.GetTagsSubscriptions(tags)
}

// RemoveNotification implements moira.Database
func (d *DatabaseWithMetrics) RemoveNotification(notificationKey string) (i1 int64, err error) {
	since := time.Now()
	if d.timers[43] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[43] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemoveNotification", "duration")
			d.timers[43] = &t
		}
	}
	defer (*d.timers[43]).UpdateSince(since)
	return d.base.RemoveNotification(notificationKey)
}

// RemovePatternWithMetrics implements moira.Database
func (d *DatabaseWithMetrics) RemovePatternWithMetrics(pattern string) (err error) {
	since := time.Now()
	if d.timers[44] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[44] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemovePatternWithMetrics", "duration")
			d.timers[44] = &t
		}
	}
	defer (*d.timers[44]).UpdateSince(since)
	return d.base.RemovePatternWithMetrics(pattern)
}

// GetTriggerLastCheck implements moira.Database
func (d *DatabaseWithMetrics) GetTriggerLastCheck(triggerID string) (c1 moira.CheckData, err error) {
	since := time.Now()
	if d.timers[45] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[45] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetTriggerLastCheck", "duration")
			d.timers[45] = &t
		}
	}
	defer (*d.timers[45]).UpdateSince(since)
	return d.base.GetTriggerLastCheck(triggerID)
}

// RemoveTriggerLastCheck implements moira.Database
func (d *DatabaseWithMetrics) RemoveTriggerLastCheck(triggerID string) (err error) {
	since := time.Now()
	if d.timers[46] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[46] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemoveTriggerLastCheck", "duration")
			d.timers[46] = &t
		}
	}
	defer (*d.timers[46]).UpdateSince(since)
	return d.base.RemoveTriggerLastCheck(triggerID)
}

// SubscribeMetricEvents implements moira.Database
func (d *DatabaseWithMetrics) SubscribeMetricEvents(tomb *tomb.Tomb) (ch1 <-chan *moira.MetricEvent, err error) {
	since := time.Now()
	if d.timers[47] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[47] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "SubscribeMetricEvents", "duration")
			d.timers[47] = &t
		}
	}
	defer (*d.timers[47]).UpdateSince(since)
	return d.base.SubscribeMetricEvents(tomb)
}

// GetMetricRetention implements moira.Database
func (d *DatabaseWithMetrics) GetMetricRetention(metric string) (i1 int64, err error) {
	since := time.Now()
	if d.timers[48] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[48] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetMetricRetention", "duration")
			d.timers[48] = &t
		}
	}
	defer (*d.timers[48]).UpdateSince(since)
	return d.base.GetMetricRetention(metric)
}

// AcquireTriggerCheckLock implements moira.Database
func (d *DatabaseWithMetrics) AcquireTriggerCheckLock(triggerID string, timeout int) (err error) {
	since := time.Now()
	if d.timers[49] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[49] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "AcquireTriggerCheckLock", "duration")
			d.timers[49] = &t
		}
	}
	defer (*d.timers[49]).UpdateSince(since)
	return d.base.AcquireTriggerCheckLock(triggerID, timeout)
}

// RemoveTriggersToReindex implements moira.Database
func (d *DatabaseWithMetrics) RemoveTriggersToReindex(to int64) (err error) {
	since := time.Now()
	if d.timers[50] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[50] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemoveTriggersToReindex", "duration")
			d.timers[50] = &t
		}
	}
	defer (*d.timers[50]).UpdateSince(since)
	return d.base.RemoveTriggersToReindex(to)
}

// GetUnusedTriggerIDs implements moira.Database
func (d *DatabaseWithMetrics) GetUnusedTriggerIDs() (sa1 []string, err error) {
	since := time.Now()
	if d.timers[51] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[51] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetUnusedTriggerIDs", "duration")
			d.timers[51] = &t
		}
	}
	defer (*d.timers[51]).UpdateSince(since)
	return d.base.GetUnusedTriggerIDs()
}

// GetTagTriggerIDs implements moira.Database
func (d *DatabaseWithMetrics) GetTagTriggerIDs(tagName string) (sa1 []string, err error) {
	since := time.Now()
	if d.timers[52] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[52] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetTagTriggerIDs", "duration")
			d.timers[52] = &t
		}
	}
	defer (*d.timers[52]).UpdateSince(since)
	return d.base.GetTagTriggerIDs(tagName)
}

// GetAllTriggerIDs implements moira.Database
func (d *DatabaseWithMetrics) GetAllTriggerIDs() (sa1 []string, err error) {
	since := time.Now()
	if d.timers[53] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[53] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetAllTriggerIDs", "duration")
			d.timers[53] = &t
		}
	}
	defer (*d.timers[53]).UpdateSince(since)
	return d.base.GetAllTriggerIDs()
}

// SetTriggerThrottling implements moira.Database
func (d *DatabaseWithMetrics) SetTriggerThrottling(triggerID string, next time.Time) (err error) {
	since := time.Now()
	if d.timers[54] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[54] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "SetTriggerThrottling", "duration")
			d.timers[54] = &t
		}
	}
	defer (*d.timers[54]).UpdateSince(since)
	return d.base.SetTriggerThrottling(triggerID, next)
}

// GetAllContacts implements moira.Database
func (d *DatabaseWithMetrics) GetAllContacts() (cpa1 []*moira.ContactData, err error) {
	since := time.Now()
	if d.timers[55] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[55] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetAllContacts", "duration")
			d.timers[55] = &t
		}
	}
	defer (*d.timers[55]).UpdateSince(since)
	return d.base.GetAllContacts()
}

// RemoveContact implements moira.Database
func (d *DatabaseWithMetrics) RemoveContact(contactID string) (err error) {
	since := time.Now()
	if d.timers[56] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[56] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemoveContact", "duration")
			d.timers[56] = &t
		}
	}
	defer (*d.timers[56]).UpdateSince(since)
	return d.base.RemoveContact(contactID)
}

// RemovePatternsMetrics implements moira.Database
func (d *DatabaseWithMetrics) RemovePatternsMetrics(pattern []string) (err error) {
	since := time.Now()
	if d.timers[57] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[57] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemovePatternsMetrics", "duration")
			d.timers[57] = &t
		}
	}
	defer (*d.timers[57]).UpdateSince(since)
	return d.base.RemovePatternsMetrics(pattern)
}

// RemoveUser implements moira.Database
func (d *DatabaseWithMetrics) RemoveUser(messenger string, username string) (err error) {
	since := time.Now()
	if d.timers[58] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[58] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemoveUser", "duration")
			d.timers[58] = &t
		}
	}
	defer (*d.timers[58]).UpdateSince(since)
	return d.base.RemoveUser(messenger, username)
}

// RemoveTag implements moira.Database
func (d *DatabaseWithMetrics) RemoveTag(tagName string) (err error) {
	since := time.Now()
	if d.timers[59] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[59] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemoveTag", "duration")
			d.timers[59] = &t
		}
	}
	defer (*d.timers[59]).UpdateSince(since)
	return d.base.RemoveTag(tagName)
}

// GetSubscription implements moira.Database
func (d *DatabaseWithMetrics) GetSubscription(id string) (s1 moira.SubscriptionData, err error) {
	since := time.Now()
	if d.timers[60] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[60] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetSubscription", "duration")
			d.timers[60] = &t
		}
	}
	defer (*d.timers[60]).UpdateSince(since)
	return d.base.GetSubscription(id)
}

// GetSubscriptions implements moira.Database
func (d *DatabaseWithMetrics) GetSubscriptions(subscriptionIDs []string) (spa1 []*moira.SubscriptionData, err error) {
	since := time.Now()
	if d.timers[61] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[61] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetSubscriptions", "duration")
			d.timers[61] = &t
		}
	}
	defer (*d.timers[61]).UpdateSince(since)
	return d.base.GetSubscriptions(subscriptionIDs)
}

// GetNotifications implements moira.Database
func (d *DatabaseWithMetrics) GetNotifications(start int64, end int64) (spa1 []*moira.ScheduledNotification, i1 int64, err error) {
	since := time.Now()
	if d.timers[62] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[62] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetNotifications", "duration")
			d.timers[62] = &t
		}
	}
	defer (*d.timers[62]).UpdateSince(since)
	return d.base.GetNotifications(start, end)
}

// GetPatterns implements moira.Database
func (d *DatabaseWithMetrics) GetPatterns() (sa1 []string, err error) {
	since := time.Now()
	if d.timers[63] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[63] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetPatterns", "duration")
			d.timers[63] = &t
		}
	}
	defer (*d.timers[63]).UpdateSince(since)
	return d.base.GetPatterns()
}

// AddLocalTriggersToCheck implements moira.Database
func (d *DatabaseWithMetrics) AddLocalTriggersToCheck(triggerIDs []string) (err error) {
	since := time.Now()
	if d.timers[64] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[64] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "AddLocalTriggersToCheck", "duration")
			d.timers[64] = &t
		}
	}
	defer (*d.timers[64]).UpdateSince(since)
	return d.base.AddLocalTriggersToCheck(triggerIDs)
}

// GetLocalTriggersToCheck implements moira.Database
func (d *DatabaseWithMetrics) GetLocalTriggersToCheck(count int) (sa1 []string, err error) {
	since := time.Now()
	if d.timers[65] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[65] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetLocalTriggersToCheck", "duration")
			d.timers[65] = &t
		}
	}
	defer (*d.timers[65]).UpdateSince(since)
	return d.base.GetLocalTriggersToCheck(count)
}

// GetRemoteChecksUpdatesCount implements moira.Database
func (d *DatabaseWithMetrics) GetRemoteChecksUpdatesCount() (i1 int64, err error) {
	since := time.Now()
	if d.timers[66] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[66] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetRemoteChecksUpdatesCount", "duration")
			d.timers[66] = &t
		}
	}
	defer (*d.timers[66]).UpdateSince(since)
	return d.base.GetRemoteChecksUpdatesCount()
}

// FetchNotificationEvent implements moira.Database
func (d *DatabaseWithMetrics) FetchNotificationEvent() (n1 moira.NotificationEvent, err error) {
	since := time.Now()
	if d.timers[67] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[67] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "FetchNotificationEvent", "duration")
			d.timers[67] = &t
		}
	}
	defer (*d.timers[67]).UpdateSince(since)
	return d.base.FetchNotificationEvent()
}

// GetUserContactIDs implements moira.Database
func (d *DatabaseWithMetrics) GetUserContactIDs(userLogin string) (sa1 []string, err error) {
	since := time.Now()
	if d.timers[68] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[68] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetUserContactIDs", "duration")
			d.timers[68] = &t
		}
	}
	defer (*d.timers[68]).UpdateSince(since)
	return d.base.GetUserContactIDs(userLogin)
}

// RemoveAllNotifications implements moira.Database
func (d *DatabaseWithMetrics) RemoveAllNotifications() (err error) {
	since := time.Now()
	if d.timers[69] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[69] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemoveAllNotifications", "duration")
			d.timers[69] = &t
		}
	}
	defer (*d.timers[69]).UpdateSince(since)
	return d.base.RemoveAllNotifications()
}

// GetMetricsValues implements moira.Database
func (d *DatabaseWithMetrics) GetMetricsValues(metrics []string, from int64, until int64) (m1 map[string][]*moira.MetricValue, err error) {
	since := time.Now()
	if d.timers[70] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[70] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetMetricsValues", "duration")
			d.timers[70] = &t
		}
	}
	defer (*d.timers[70]).UpdateSince(since)
	return d.base.GetMetricsValues(metrics, from, until)
}

// RemoveMetricsValues implements moira.Database
func (d *DatabaseWithMetrics) RemoveMetricsValues(metrics []string, toTime int64) (err error) {
	since := time.Now()
	if d.timers[71] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[71] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemoveMetricsValues", "duration")
			d.timers[71] = &t
		}
	}
	defer (*d.timers[71]).UpdateSince(since)
	return d.base.RemoveMetricsValues(metrics, toTime)
}

// NewLock implements moira.Database
func (d *DatabaseWithMetrics) NewLock(name string, ttl time.Duration) (l1 moira.Lock) {
	since := time.Now()
	if d.timers[72] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[72] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "NewLock", "duration")
			d.timers[72] = &t
		}
	}
	defer (*d.timers[72]).UpdateSince(since)
	return d.base.NewLock(name, ttl)
}

// GetChecksUpdatesCount implements moira.Database
func (d *DatabaseWithMetrics) GetChecksUpdatesCount() (i1 int64, err error) {
	since := time.Now()
	if d.timers[73] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[73] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetChecksUpdatesCount", "duration")
			d.timers[73] = &t
		}
	}
	defer (*d.timers[73]).UpdateSince(since)
	return d.base.GetChecksUpdatesCount()
}

// GetPatternTriggerIDs implements moira.Database
func (d *DatabaseWithMetrics) GetPatternTriggerIDs(pattern string) (sa1 []string, err error) {
	since := time.Now()
	if d.timers[74] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[74] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetPatternTriggerIDs", "duration")
			d.timers[74] = &t
		}
	}
	defer (*d.timers[74]).UpdateSince(since)
	return d.base.GetPatternTriggerIDs(pattern)
}

// RemovePatternTriggerIDs implements moira.Database
func (d *DatabaseWithMetrics) RemovePatternTriggerIDs(pattern string) (err error) {
	since := time.Now()
	if d.timers[75] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[75] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "RemovePatternTriggerIDs", "duration")
			d.timers[75] = &t
		}
	}
	defer (*d.timers[75]).UpdateSince(since)
	return d.base.RemovePatternTriggerIDs(pattern)
}

// GetContacts implements moira.Database
func (d *DatabaseWithMetrics) GetContacts(contactIDs []string) (cpa1 []*moira.ContactData, err error) {
	since := time.Now()
	if d.timers[76] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[76] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetContacts", "duration")
			d.timers[76] = &t
		}
	}
	defer (*d.timers[76]).UpdateSince(since)
	return d.base.GetContacts(contactIDs)
}

// AddNotifications implements moira.Database
func (d *DatabaseWithMetrics) AddNotifications(notification []*moira.ScheduledNotification, timestamp int64) (err error) {
	since := time.Now()
	if d.timers[77] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[77] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "AddNotifications", "duration")
			d.timers[77] = &t
		}
	}
	defer (*d.timers[77]).UpdateSince(since)
	return d.base.AddNotifications(notification, timestamp)
}

// SetTriggerCheckLock implements moira.Database
func (d *DatabaseWithMetrics) SetTriggerCheckLock(triggerID string) (b1 bool, err error) {
	since := time.Now()
	if d.timers[78] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[78] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "SetTriggerCheckLock", "duration")
			d.timers[78] = &t
		}
	}
	defer (*d.timers[78]).UpdateSince(since)
	return d.base.SetTriggerCheckLock(triggerID)
}

// GetTriggers implements moira.Database
func (d *DatabaseWithMetrics) GetTriggers(triggerIDs []string) (tpa1 []*moira.Trigger, err error) {
	since := time.Now()
	if d.timers[79] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[79] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetTriggers", "duration")
			d.timers[79] = &t
		}
	}
	defer (*d.timers[79]).UpdateSince(since)
	return d.base.GetTriggers(triggerIDs)
}

// GetTriggersSearchResults implements moira.Database
func (d *DatabaseWithMetrics) GetTriggersSearchResults(searchResultsID string, page int64, size int64) (spa1 []*moira.SearchResult, i1 int64, err error) {
	since := time.Now()
	if d.timers[80] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[80] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetTriggersSearchResults", "duration")
			d.timers[80] = &t
		}
	}
	defer (*d.timers[80]).UpdateSince(since)
	return d.base.GetTriggersSearchResults(searchResultsID, page, size)
}

// GetMetricsUpdatesCount implements moira.Database
func (d *DatabaseWithMetrics) GetMetricsUpdatesCount() (i1 int64, err error) {
	since := time.Now()
	if d.timers[81] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[81] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetMetricsUpdatesCount", "duration")
			d.timers[81] = &t
		}
	}
	defer (*d.timers[81]).UpdateSince(since)
	return d.base.GetMetricsUpdatesCount()
}

// PushNotificationEvent implements moira.Database
func (d *DatabaseWithMetrics) PushNotificationEvent(event *moira.NotificationEvent, ui bool) (err error) {
	since := time.Now()
	if d.timers[82] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[82] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "PushNotificationEvent", "duration")
			d.timers[82] = &t
		}
	}
	defer (*d.timers[82]).UpdateSince(since)
	return d.base.PushNotificationEvent(event, ui)
}

// GetRemoteTriggersToCheck implements moira.Database
func (d *DatabaseWithMetrics) GetRemoteTriggersToCheck(count int) (sa1 []string, err error) {
	since := time.Now()
	if d.timers[83] == nil {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.timers[83] == nil {
			t := (*d.registry).NewTimer(d.metricsPrefix, "Database", "method", "GetRemoteTriggersToCheck", "duration")
			d.timers[83] = &t
		}
	}
	defer (*d.timers[83]).UpdateSince(since)
	return d.base.GetRemoteTriggersToCheck(count)
}
