package selfstate

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
	w "github.com/moira-alert/moira/worker"
)

var defaultCheckInterval = time.Second * 10

const (
	redisDisconnectedErrorMessage  = "Redis disconnected"
	filterStateErrorMessage        = "Moira-Filter does not receive metrics"
	checkerStateErrorMessage       = "Moira-Checker does not check triggers"
	remoteCheckerStateErrorMessage = "Moira-Remote-Checker does not check remote triggers"
)

const selfStateLockName = "moira-self-state-monitor"
const selfStateLockTTL = time.Second * 15

// SelfCheckWorker checks what all notifier services works correctly and send message when moira don't work
type SelfCheckWorker struct {
	Log      moira.Logger
	DB       moira.Database
	Notifier notifier.Notifier
	Config   Config
	tomb     tomb.Tomb
}

func (selfCheck *SelfCheckWorker) selfStateChecker(stop <-chan struct{}) {
	if !selfCheck.Config.Enabled {
		selfCheck.Log.Debugf("Moira Self State Monitoring disabled")
		return
	}
	senders := selfCheck.Notifier.GetSenders()
	if err := selfCheck.Config.checkConfig(senders); err != nil {
		selfCheck.Log.Errorf("Can't configure Moira Self State Monitoring: %s", err.Error())
		return
	}

	selfCheck.Log.Info("Moira Notifier Self State Monitor started")

	var metricsCount, checksCount, remoteChecksCount int64
	lastMetricReceivedTS := time.Now().Unix()
	redisLastCheckTS := time.Now().Unix()
	lastCheckTS := time.Now().Unix()
	lastRemoteCheckTS := time.Now().Unix()
	nextSendErrorMessage := time.Now().Unix()

	checkTicker := time.NewTicker(defaultCheckInterval)
	defer checkTicker.Stop()

	for {
		select {
		case <-stop:
			selfCheck.Log.Info("Moira Notifier Self State Monitor stopped")
			return
		case <-checkTicker.C:
			selfCheck.check(time.Now().Unix(), &lastMetricReceivedTS, &redisLastCheckTS, &lastCheckTS, &lastRemoteCheckTS, &nextSendErrorMessage, &metricsCount, &checksCount, &remoteChecksCount)
		}
	}
}

// Start self check worker
func (selfCheck *SelfCheckWorker) Start() error {

	selfCheck.tomb.Go(func() error {
		w.NewWorker(
			"Moira Self State Monitoring",
			selfCheck.Log,
			selfCheck.DB.NewLock(selfStateLockName, selfStateLockTTL),
			selfCheck.selfStateChecker,
		).Run(selfCheck.tomb.Dying())
		return nil
	})

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

func (selfCheck *SelfCheckWorker) check(nowTS int64, lastMetricReceivedTS, redisLastCheckTS, lastCheckTS, lastRemoteCheckTS, nextSendErrorMessage, metricsCount, checksCount, remoteChecksCount *int64) {
	var events []moira.NotificationEvent
	var rcc int64

	mc, _ := selfCheck.DB.GetMetricsUpdatesCount()
	cc, err := selfCheck.DB.GetChecksUpdatesCount()
	if selfCheck.Config.RemoteTriggersEnabled {
		rcc, _ = selfCheck.DB.GetRemoteChecksUpdatesCount()
	}
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
		if selfCheck.Config.RemoteTriggersEnabled {
			if *remoteChecksCount != rcc {
				*remoteChecksCount = rcc
				*lastRemoteCheckTS = nowTS
			}
		}
	}

	if *nextSendErrorMessage < nowTS {
		if *redisLastCheckTS < nowTS-selfCheck.Config.RedisDisconnectDelaySeconds {
			interval := nowTS - *redisLastCheckTS
			selfCheck.Log.Errorf("%s more than %ds. Send message.", redisDisconnectedErrorMessage, interval)
			appendNotificationEvents(&events, redisDisconnectedErrorMessage, interval)
		}

		if *lastMetricReceivedTS < nowTS-selfCheck.Config.LastMetricReceivedDelaySeconds && err == nil {
			interval := nowTS - *lastMetricReceivedTS
			selfCheck.Log.Errorf("%s more than %ds. Send message.", filterStateErrorMessage, interval)
			appendNotificationEvents(&events, filterStateErrorMessage, interval)
			selfCheck.setNotifierState(ERROR)
		}

		if *lastCheckTS < nowTS-selfCheck.Config.LastCheckDelaySeconds && err == nil {
			interval := nowTS - *lastCheckTS
			selfCheck.Log.Errorf("%s more than %ds. Send message.", checkerStateErrorMessage, interval)
			appendNotificationEvents(&events, checkerStateErrorMessage, interval)
			selfCheck.setNotifierState(ERROR)
		}

		if selfCheck.Config.RemoteTriggersEnabled {
			if *lastRemoteCheckTS < nowTS-selfCheck.Config.LastRemoteCheckDelaySeconds && err == nil {
				interval := nowTS - *lastRemoteCheckTS
				selfCheck.Log.Errorf("%s more than %ds. Send message.", remoteCheckerStateErrorMessage, interval)
				appendNotificationEvents(&events, remoteCheckerStateErrorMessage, interval)
			}
		}

		if notifierState, _ := selfCheck.DB.GetNotifierState(); notifierState != OK {
			selfCheck.Log.Errorf("%s. Send message.", notifierStateErrorMessage(notifierState))
			appendNotificationEvents(&events, notifierStateErrorMessage(notifierState), 0)
		}

		if len(events) > 0 {
			eventsJSON, _ := json.Marshal(events)
			selfCheck.Log.Errorf("Health check. Send package of %v notification events: %s", len(events), eventsJSON)
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

func notifierStateErrorMessage(state string) string {
	const template = "Moira-Notifier does not send messages. State: %v"
	return fmt.Sprintf(template, state)
}
