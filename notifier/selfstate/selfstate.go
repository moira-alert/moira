package selfstate

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/notifier"
	"sync"
	"time"
)

type SelfCheckWorker struct {
	logger            moira_alert.Logger
	database          moira_alert.Database
	notifier          *notifier.Notifier
	config            Config
	SelfCheckInterval time.Duration
}

var DefaultCheckInterval = time.Second * 10

func Init(database moira_alert.Database, logger moira_alert.Logger, config Config, notifier2 *notifier.Notifier) (worker *SelfCheckWorker, needRun bool) {
	if err := config.Check(notifier2.GetSendersHash()); err != nil {
		logger.Fatalf("Can't configure self state monitor: %s", err.Error())
	}
	if config.Enabled {
		worker = &SelfCheckWorker{
			logger:            logger,
			database:          database,
			SelfCheckInterval: DefaultCheckInterval,
			notifier:          notifier2,
			config:            config,
		}
		needRun = true
		return worker, needRun
	}
	return nil, false
}

// Send message when moira don't work
func (selfCheck SelfCheckWorker) Run(shutdown chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	now := time.Now()

	var metricsCount, checksCount int64
	checkTicker := time.NewTicker(selfCheck.SelfCheckInterval)
	lastMetricReceivedTS := now.Unix()
	redisLastCheckTS := now.Unix()
	lastCheckTS := now.Unix()
	nextSendErrorMessage := now.Unix()

	selfCheck.logger.Debugf("Start Moira Self State Monitor")
	for {
		select {
		case <-shutdown:
			checkTicker.Stop()
			selfCheck.logger.Debugf("Stop Self State Monitor")
			return
		case <-checkTicker.C:
			nowTS := now.Unix()
			mc, _ := selfCheck.database.GetMetricsCount()
			cc, err := selfCheck.database.GetChecksCount()
			if err == nil {
				redisLastCheckTS = nowTS
				if metricsCount != mc {
					metricsCount = mc
					lastMetricReceivedTS = nowTS
				}
				if checksCount != cc {
					checksCount = cc
					lastCheckTS = nowTS
				}
			}
			if nextSendErrorMessage < nowTS {
				if redisLastCheckTS < nowTS-selfCheck.config.RedisDisconnectDelay {
					selfCheck.logger.Errorf("Redis disconnected more %ds. Send message.", nowTS-redisLastCheckTS)
					selfCheck.sendErrorMessages("Redis disconnected", nowTS-redisLastCheckTS, selfCheck.config.RedisDisconnectDelay)
					nextSendErrorMessage = nowTS + selfCheck.config.NoticeInterval
					continue
				}
				if lastMetricReceivedTS < nowTS-selfCheck.config.LastMetricReceivedDelay && err == nil {
					selfCheck.logger.Errorf("Moira-Cache does not received new metrics more %ds. Send message.", nowTS-lastMetricReceivedTS)
					selfCheck.sendErrorMessages("Moira-Cache does not received new metrics", nowTS-lastMetricReceivedTS, selfCheck.config.LastMetricReceivedDelay)
					nextSendErrorMessage = nowTS + selfCheck.config.NoticeInterval
					continue
				}
				if lastCheckTS < nowTS-selfCheck.config.LastCheckDelay && err == nil {
					selfCheck.logger.Errorf("Moira-Checker does not checks triggers more %ds. Send message.", nowTS-lastCheckTS)
					selfCheck.sendErrorMessages("Moira-Checker does not checks triggers", nowTS-lastCheckTS, selfCheck.config.LastCheckDelay)
					nextSendErrorMessage = nowTS + selfCheck.config.NoticeInterval
				}
			}
		}
	}
}

func (selfCheck *SelfCheckWorker) sendErrorMessages(message string, currentValue int64, errValue int64) {
	var sendingWG sync.WaitGroup
	for _, adminContact := range selfCheck.config.Contacts {
		pkg := notifier.NotificationPackage{
			Contact: moira_alert.ContactData{
				Type:  adminContact["type"],
				Value: adminContact["value"],
			},
			Trigger: moira_alert.TriggerData{
				Name:       message,
				ErrorValue: float64(errValue),
			},
			Events: []moira_alert.EventData{
				{
					Timestamp: time.Now().Unix(),
					State:     "ERROR",
					Metric:    message,
					Value:     float64(currentValue),
				},
			},
			DontResend: true,
		}
		selfCheck.notifier.Send(&pkg, &sendingWG)
		sendingWG.Wait()
	}
}
