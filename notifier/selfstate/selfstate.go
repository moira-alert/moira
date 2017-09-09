package selfstate

import (
	"fmt"
	"sync"
	"time"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/notifier"
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
		return fmt.Errorf("Can't configure self state monitor: %s", err.Error())
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
		if *redisLastCheckTS < nowTS-selfCheck.Config.RedisDisconnectDelay {
			interval := nowTS - *redisLastCheckTS
			selfCheck.Log.Errorf("Redis disconnected more %ds. Send message.", interval)
			selfCheck.sendErrorMessages("Redis disconnected", interval, selfCheck.Config.RedisDisconnectDelay)
			*nextSendErrorMessage = nowTS + selfCheck.Config.NoticeInterval
			return
		}
		if *lastMetricReceivedTS < nowTS-selfCheck.Config.LastMetricReceivedDelay && err == nil {
			interval := nowTS - *lastMetricReceivedTS
			selfCheck.Log.Errorf("Moira-Filter does not received new metrics more %ds. Send message.", interval)
			selfCheck.sendErrorMessages("Moira-Filter does not received new metrics", interval, selfCheck.Config.LastMetricReceivedDelay)
			*nextSendErrorMessage = nowTS + selfCheck.Config.NoticeInterval
			return
		}
		if *lastCheckTS < nowTS-selfCheck.Config.LastCheckDelay && err == nil {
			interval := nowTS - *lastCheckTS
			selfCheck.Log.Errorf("Moira-Checker does not checks triggers more %ds. Send message.", interval)
			selfCheck.sendErrorMessages("Moira-Checker does not checks triggers", interval, selfCheck.Config.LastCheckDelay)
			*nextSendErrorMessage = nowTS + selfCheck.Config.NoticeInterval
		}
	}
}

func (selfCheck *SelfCheckWorker) sendErrorMessages(message string, currentValue int64, errValue int64) {
	var sendingWG sync.WaitGroup
	for _, adminContact := range selfCheck.Config.Contacts {
		val := float64(currentValue)
		pkg := notifier.NotificationPackage{
			Contact: moira.ContactData{
				Type:  adminContact["type"],
				Value: adminContact["value"],
			},
			Trigger: moira.TriggerData{
				Name:       message,
				ErrorValue: float64(errValue),
			},
			Events: []moira.NotificationEvent{
				{
					Timestamp: time.Now().Unix(),
					State:     "ERROR",
					Metric:    message,
					Value:     &val,
				},
			},
			DontResend: true,
		}
		selfCheck.Notifier.Send(&pkg, &sendingWG)
		sendingWG.Wait()
	}
}
