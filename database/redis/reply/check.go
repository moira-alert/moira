package reply

import (
	"encoding/json"
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

//TODO(litleleprikon): START remove in moira v2.8.0. Compatibility with moira < v2.6.0
const firstTarget = "t1"

//TODO(litleleprikon): END remove in moira v2.8.0. Compatibility with moira < v2.6.0

type checkDataStorageElement struct {
	Metrics                      map[string]moira.MetricState `json:"metrics"`
	MetricsToTargetRelation      map[string]string            `json:"metrics_to_target_relation"`
	Score                        int64                        `json:"score"`
	State                        moira.State                  `json:"state"`
	Maintenance                  int64                        `json:"maintenance,omitempty"`
	MaintenanceInfo              moira.MaintenanceInfo        `json:"maintenance_info"`
	Timestamp                    int64                        `json:"timestamp,omitempty"`
	EventTimestamp               int64                        `json:"event_timestamp,omitempty"`
	LastSuccessfulCheckTimestamp int64                        `json:"last_successful_check_timestamp"`
	Suppressed                   bool                         `json:"suppressed,omitempty"`
	SuppressedState              moira.State                  `json:"suppressed_state,omitempty"`
	Message                      string                       `json:"msg,omitempty"`
}

func toCheckDataStorageElement(check moira.CheckData) checkDataStorageElement {
	//TODO(litleleprikon): START remove in moira v2.8.0. Compatibility with moira < v2.6.0
	for metricName, metricState := range check.Metrics {
		if metricState.Value == nil {
			if value, ok := metricState.Values[firstTarget]; ok {
				metricState.Value = &value
				check.Metrics[metricName] = metricState
			}
		}
	}
	//TODO(litleleprikon): END remove in moira v2.8.0. Compatibility with moira < v2.6.0
	return checkDataStorageElement{
		Metrics:                      check.Metrics,
		MetricsToTargetRelation:      check.MetricsToTargetRelation,
		Score:                        check.Score,
		State:                        check.State,
		Maintenance:                  check.Maintenance,
		MaintenanceInfo:              check.MaintenanceInfo,
		Timestamp:                    check.Timestamp,
		EventTimestamp:               check.EventTimestamp,
		LastSuccessfulCheckTimestamp: check.LastSuccessfulCheckTimestamp,
		Suppressed:                   check.Suppressed,
		SuppressedState:              check.SuppressedState,
		Message:                      check.Message,
	}
}

func (d checkDataStorageElement) toCheckData() moira.CheckData {
	//TODO(litleleprikon): START remove in moira v2.8.0. Compatibility with moira < v2.6.0
	for metricName, metricState := range d.Metrics {
		if metricState.Values == nil {
			metricState.Values = make(map[string]float64)
		}
		if metricState.Value != nil {
			metricState.Values[firstTarget] = *metricState.Value
			metricState.Value = nil
		}
		d.Metrics[metricName] = metricState
	}
	if d.MetricsToTargetRelation == nil {
		d.MetricsToTargetRelation = make(map[string]string)
	}
	//TODO(litleleprikon): END remove in moira v2.8.0. Compatibility with moira < v2.6.0
	return moira.CheckData{
		Metrics:                      d.Metrics,
		MetricsToTargetRelation:      d.MetricsToTargetRelation,
		Score:                        d.Score,
		State:                        d.State,
		Maintenance:                  d.Maintenance,
		MaintenanceInfo:              d.MaintenanceInfo,
		Timestamp:                    d.Timestamp,
		EventTimestamp:               d.EventTimestamp,
		LastSuccessfulCheckTimestamp: d.LastSuccessfulCheckTimestamp,
		Suppressed:                   d.Suppressed,
		SuppressedState:              d.SuppressedState,
		Message:                      d.Message,
	}
}

// Check converts redis DB reply to moira.CheckData
func Check(rep interface{}, err error) (moira.CheckData, error) {
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return moira.CheckData{}, database.ErrNil
		}
		return moira.CheckData{}, fmt.Errorf("failed to read lastCheck: %s", err.Error())
	}
	checkSE := checkDataStorageElement{}
	err = json.Unmarshal(bytes, &checkSE)
	if err != nil {
		return moira.CheckData{}, fmt.Errorf("failed to parse lastCheck json %s: %s", string(bytes), err.Error())
	}
	return checkSE.toCheckData(), nil
}

// GetCheckBytes is a function that takes moira.CheckData and turns it to bytes that will be saved in redis.
func GetCheckBytes(check moira.CheckData) ([]byte, error) {
	checkSE := toCheckDataStorageElement(check)
	bytes, err := json.Marshal(checkSE)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal check data: %s", err.Error())
	}
	return bytes, nil
}
