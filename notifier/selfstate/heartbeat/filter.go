package heartbeat

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/moira-alert/moira"
)

var (
	localClusterKey = moira.DefaultLocalCluster

	_ Heartbeater = (*filterHeartbeater)(nil)
)

type FilterHeartbeaterConfig struct {
	HeartbeaterBaseConfig

	MetricReceivedDelay time.Duration `validate:"required,gt=0"`
}

func (cfg FilterHeartbeaterConfig) validate() error {
	validator := validator.New()
	return validator.Struct(cfg)
}

type filterHeartbeater struct {
	*heartbeaterBase

	cfg              FilterHeartbeaterConfig
	lastMetricsCount int64
}

func NewFilterHeartbeater(cfg FilterHeartbeaterConfig, base *heartbeaterBase) (*filterHeartbeater, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("filter heartheater configuration error: %w", err)
	}

	return &filterHeartbeater{
		heartbeaterBase: base,
		cfg:             cfg,
	}, nil
}

func (heartbeater *filterHeartbeater) Check() (State, error) {
	triggersCount, err := heartbeater.database.GetTriggersToCheckCount(localClusterKey)
	if err != nil {
		return StateError, err
	}

	metricsCount, err := heartbeater.database.GetMetricsUpdatesCount()
	if err != nil {
		return StateError, err
	}

	now := heartbeater.clock.NowUTC()
	if heartbeater.lastMetricsCount != metricsCount || triggersCount == 0 {
		heartbeater.lastMetricsCount = metricsCount
		heartbeater.lastSuccessfulCheck = now
		return StateOK, nil
	}

	if now.Sub(heartbeater.lastSuccessfulCheck) > heartbeater.cfg.MetricReceivedDelay {
		return StateError, nil
	}

	return StateOK, nil
}

// NeedTurnOffNotifier: turn off notifications if at least once the filter check was successful.
func (heartbeater filterHeartbeater) NeedTurnOffNotifier() bool {
	return heartbeater.cfg.NeedTurnOffNotifier
}

func (filterHeartbeater) Type() moira.EmergencyContactType {
	return moira.EmergencyTypeFilterNoMetricsReceived
}

func (heartbeater filterHeartbeater) AlertSettings() AlertConfig {
	return heartbeater.cfg.AlertCfg
}
