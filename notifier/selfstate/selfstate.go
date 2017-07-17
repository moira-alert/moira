package selfstate

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/notifier"
	"sync"
	"time"
)

//SelfCheckWorker - check what all notifier services works correctly and send message when moira don't work
type SelfCheckWorker struct {
	logger            moira.Logger
	database          moira.Database
	notifier          notifier.Notifier
	config            Config
	selfCheckInterval time.Duration
}

var defaultCheckInterval = time.Second * 10

//NewSelfCheckWorker - initialize notifier self check worker
func NewSelfCheckWorker(database moira.Database, logger moira.Logger, config Config, notifier2 notifier.Notifier) (worker *SelfCheckWorker, needRun bool) {
	senders := notifier2.GetSenders()
	if err := config.checkConfig(senders); err != nil {
		logger.Fatalf("Can't configure self state monitor: %s", err.Error())
	}
	if config.Enabled {
		worker = &SelfCheckWorker{
			logger:            logger,
			database:          database,
			selfCheckInterval: defaultCheckInterval,
			notifier:          notifier2,
			config:            config,
		}
		needRun = true
		return worker, needRun
	}
	return nil, false
}

//Run self check worker
func (selfCheck SelfCheckWorker) Run(shutdown chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	var metricsCount, checksCount int64
	checkTicker := time.NewTicker(selfCheck.selfCheckInterval)
	lastMetricReceivedTS := time.Now().Unix()
	redisLastCheckTS := time.Now().Unix()
	lastCheckTS := time.Now().Unix()
	nextSendErrorMessage := time.Now().Unix()

	selfCheck.logger.Debugf("Start Moira Self State Monitor")
	for {
		select {
		case <-shutdown:
			checkTicker.Stop()
			selfCheck.logger.Debugf("Stop Self State Monitor")
			return
		case <-checkTicker.C:
			selfCheck.check(time.Now().Unix(), &lastMetricReceivedTS, &redisLastCheckTS, &lastCheckTS, &nextSendErrorMessage, &metricsCount, &checksCount)
		}
	}
}

func (selfCheck *SelfCheckWorker) check(nowTS int64, lastMetricReceivedTS, redisLastCheckTS, lastCheckTS, nextSendErrorMessage, metricsCount, checksCount *int64) {
	mc, _ := selfCheck.database.GetMetricsCount()
	cc, err := selfCheck.database.GetChecksCount()
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
		if *redisLastCheckTS < nowTS-selfCheck.config.RedisDisconnectDelay {
			interval := nowTS - *redisLastCheckTS
			selfCheck.logger.Errorf("Redis disconnected more %ds. Send message.", interval)
			selfCheck.sendErrorMessages("Redis disconnected", interval, selfCheck.config.RedisDisconnectDelay)
			*nextSendErrorMessage = nowTS + selfCheck.config.NoticeInterval
			return
		}
		if *lastMetricReceivedTS < nowTS-selfCheck.config.LastMetricReceivedDelay && err == nil {
			interval := nowTS - *lastMetricReceivedTS
			selfCheck.logger.Errorf("Moira-Cache does not received new metrics more %ds. Send message.", interval)
			selfCheck.sendErrorMessages("Moira-Cache does not received new metrics", interval, selfCheck.config.LastMetricReceivedDelay)
			*nextSendErrorMessage = nowTS + selfCheck.config.NoticeInterval
			return
		}
		if *lastCheckTS < nowTS-selfCheck.config.LastCheckDelay && err == nil {
			interval := nowTS - *lastCheckTS
			selfCheck.logger.Errorf("Moira-Checker does not checks triggers more %ds. Send message.", interval)
			selfCheck.sendErrorMessages("Moira-Checker does not checks triggers", interval, selfCheck.config.LastCheckDelay)
			*nextSendErrorMessage = nowTS + selfCheck.config.NoticeInterval
		}
	}
}

func (selfCheck *SelfCheckWorker) sendErrorMessages(message string, currentValue int64, errValue int64) {
	var sendingWG sync.WaitGroup
	for _, adminContact := range selfCheck.config.Contacts {
		pkg := notifier.NotificationPackage{
			Contact: moira.ContactData{
				Type:  adminContact["type"],
				Value: adminContact["value"],
			},
			Trigger: moira.TriggerData{
				Name:       message,
				ErrorValue: float64(errValue),
			},
			Events: []moira.EventData{
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
