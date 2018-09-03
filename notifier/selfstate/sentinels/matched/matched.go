package matched

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gosexy/to"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
)

type Sentinel struct {
	database       moira.Database
	logger         moira.Logger
	matchedEpsilon time.Duration
	matchedK       int64
	ttl            time.Duration
	tomb           tomb.Tomb
}

func (sentinel *Sentinel) Init(sentinelSettings map[string]string, database moira.Database, logger moira.Logger) error {
	var err error
	sentinel.database = database
	sentinel.logger = logger
	sentinel.matchedEpsilon = to.Duration(sentinelSettings["epsilon"])
	if sentinel.matchedEpsilon == time.Duration(0) {
		return fmt.Errorf("can not read sentinel matched epsilon from config")
	}
	sentinel.matchedK, err = strconv.ParseInt(sentinelSettings["k"], 10, 64)
	if err != nil {
		return fmt.Errorf("can not read sentinel matched k from config: %s", err.Error())
	}
	sentinel.ttl = to.Duration(sentinelSettings["ttl"])
	if sentinel.ttl == time.Duration(0) {
		return fmt.Errorf("can not read sentinel ttl from config")
	}
	return nil
}

func (sentinel *Sentinel) Protect() error {
	var matchedDelta *int64
	var matchedCount *int64
	startMatchedCount, err := sentinel.database.GetMatchedMetricsUpdatesCount()
	if err != nil {
		sentinel.logger.Infof("Get matched state failed: %s", err.Error())
	} else {
		matchedCount = &startMatchedCount
	}

	sentinel.tomb.Go(func() error {
		protectTicker := time.NewTicker(sentinel.matchedEpsilon)
		for {
			select {
			case <- sentinel.tomb.Dying():
				sentinel.logger.Infof("Sentinel stopped")
				return nil
			case <- protectTicker.C:
				newMatchedCount, err := sentinel.database.GetMatchedMetricsUpdatesCount()
				if err != nil {
					sentinel.logger.Infof("Get matched state failed: %s", err.Error())
				} else {
					if matchedCount != nil {
						newMatchedDelta := newMatchedCount - *matchedCount
						if matchedDelta != nil {
							if newMatchedDelta < (*matchedDelta * sentinel.matchedK) {
								degraded := sentinel.Inspect(*matchedDelta, newMatchedDelta)
								if degraded {
									time.Sleep(sentinel.ttl)
								}
							}
						}
						matchedDelta = &newMatchedDelta
					}
					matchedCount = &newMatchedCount
				}
			}
		}
	})
	sentinel.logger.Info("Sentinel started")
	return nil
}

func (sentinel *Sentinel) Inspect(oldValue int64, newValue int64) bool {
	state, err := sentinel.database.GetNotifierState()
	if err != nil {
		sentinel.logger.Infof("Failed to get Notifier state: %s", err.Error())
		return false
	}
	degraded := oldValue > (newValue * sentinel.matchedK)
	if degraded && state == "OK" {
		sentinel.logger.Infof(
			"Matched state degraded. Old value: %d, new value: %d. Disabling Notifier",
			oldValue, newValue)
		sentinel.database.SetNotifierState("ERROR")
	}
	if !degraded && state == "ERROR" {
		sentinel.logger.Infof(
			"Matched state recovered. Old value: %d, new value: %d. Enabling Notifier",
			oldValue, newValue)
		sentinel.database.SetNotifierState("OK")
	}
	return degraded
}
