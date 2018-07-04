package selfstate

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
)

var defaultCheckInterval = time.Second * 10

// SelfCheckWorker checks what all notifier services works correctly and send message when moira don't work
type SelfCheckWorker struct {
	Log      moira.Logger
	DB       moira.Database
	Notifier notifier.Notifier
	Config   Config
	tomb     tomb.Tomb
}

// Start self check worker
func (selfCheck *SelfCheckWorker) Start() error {
	if !selfCheck.Config.Enabled {
		selfCheck.Log.Debugf("Moira Self State Monitoring disabled")
		return nil
	}
	senders := selfCheck.Notifier.GetSenders()
	if err := selfCheck.Config.checkConfig(senders); err != nil {
		return fmt.Errorf("can't configure self state monitor: %s", err.Error())
	}
	var metricsCount, checksCount int64
	lastMetricReceivedTS := time.Now().Unix()
	redisLastCheckTS := time.Now().Unix()
	lastCheckTS := time.Now().Unix()
	nextSendErrorMessage := time.Now().Unix()

	selfCheck.tomb.Go(func() error {
		checkTicker := time.NewTicker(defaultCheckInterval)
		for {
			select {
			case <-selfCheck.tomb.Dying():
				checkTicker.Stop()
				selfCheck.Log.Info("Moira Notifier Self State Monitor Stopped")
				return nil
			case <-checkTicker.C:
				selfCheck.check(time.Now().Unix(), &lastMetricReceivedTS, &redisLastCheckTS, &lastCheckTS, &nextSendErrorMessage, &metricsCount, &checksCount)
			}
		}
	})

	selfCheck.Log.Info("Moira Notifier Self State Monitor Started")
	return nil
}

// Stop self check worker and wait for finish
func (selfCheck *SelfCheckWorker) Stop() error {
	if !selfCheck.Config.Enabled {
		return nil
	}
	selfCheck.tomb.Kill(nil)
	return selfCheck.tomb.Wait()
}

func (selfCheck *SelfCheckWorker) check(nowTS int64, lastMetricReceivedTS, redisLastCheckTS, lastCheckTS, nextSendErrorMessage, metricsCount, checksCount *int64) {
	var events []moira.NotificationEvent

	mc, _ := selfCheck.DB.GetMetricsUpdatesCount()
	cc, err := selfCheck.DB.GetChecksUpdatesCount()
	if err == nil {
		*redisLastCheckTS = nowTS
		if *metricsCount != mc {
			*metricsCount = mc
			*lastMetricReceivedTS = nowTS
		}
		if *checksCount != cc {
			*checksCount = cc
			*lastCheckTS = nowTS
		}
	}

	if *nextSendErrorMessage < nowTS {
		if *redisLastCheckTS < nowTS-selfCheck.Config.RedisDisconnectDelaySeconds {
			interval := nowTS - *redisLastCheckTS
			selfCheck.Log.Errorf("Redis disconnected more %ds. Send message.", interval)
			appendNotificationEvents(&events, "Redis disconnected", interval)
		}

		if *lastMetricReceivedTS < nowTS-selfCheck.Config.LastMetricReceivedDelaySeconds && err == nil {
			interval := nowTS - *lastMetricReceivedTS
			selfCheck.Log.Errorf("Moira-Filter does not received new metrics more %ds. Send message.", interval)
			appendNotificationEvents(&events, "Moira-Filter does not received new metrics", interval)
			selfCheck.setNotifierState(ERROR)
		}

		if *lastCheckTS < nowTS-selfCheck.Config.LastCheckDelaySeconds && err == nil {
			interval := nowTS - *lastCheckTS
			selfCheck.Log.Errorf("Moira-Checker does not checks triggers more %ds. Send message.", interval)
			appendNotificationEvents(&events, "Moira-Checker does not checks triggers", interval)
			selfCheck.setNotifierState(ERROR)
		}

		if notifierState, _ := selfCheck.DB.GetNotifierState(); notifierState != OK {
			selfCheck.Log.Errorf("Notifier state: %v. Send message.", notifierState)
			message := fmt.Sprintf("Notifier state: %v. Events are not sending to recipients", notifierState)
			appendNotificationEvents(&events, message, 0)
		}

		if len(events) > 0 {
			eventsJson, _ := json.Marshal(events)
			selfCheck.Log.Errorf("Selfstate check. Send package of %v notification events: %s", len(events), eventsJson)
			selfCheck.sendErrorMessages(&events)
			*nextSendErrorMessage = nowTS + selfCheck.Config.NoticeIntervalSeconds
		}

	}
}

func appendNotificationEvents(events *[]moira.NotificationEvent, message string, currentValue int64) {
	val := float64(currentValue)
	event := moira.NotificationEvent{
		Timestamp: time.Now().Unix(),
		OldState:  "NODATA",
		State:     "ERROR",
		Metric:    message,
		Value:     &val,
	}

	*events = append(*events, event)
}

func (selfCheck *SelfCheckWorker) sendErrorMessages(events *[]moira.NotificationEvent) {
	var sendingWG sync.WaitGroup
	for _, adminContact := range selfCheck.Config.Contacts {
		pkg := notifier.NotificationPackage{
			Contact: moira.ContactData{
				Type:  adminContact["type"],
				Value: adminContact["value"],
			},
			Trigger: moira.TriggerData{
				Name:       "Moira health check",
				ErrorValue: float64(0),
			},
			Events:     *events,
			DontResend: true,
		}
		selfCheck.Notifier.Send(&pkg, &sendingWG)
		sendingWG.Wait()
	}
}

func (selfCheck *SelfCheckWorker) setNotifierState(state string) {
	err := selfCheck.DB.SetNotifierState(state)
	if err != nil {
		selfCheck.Log.Errorf("Can't set notifier state: %v", err)
	}
}
