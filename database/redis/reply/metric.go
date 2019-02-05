package reply

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gomodule/redigo/redis"
	"github.com/moira-alert/moira"
)

// MetricValues converts redis DB reply struct "RetentionTimestamp Value" "Timestamp" to moira.MetricValue object
func MetricValues(values interface{}) ([]*moira.MetricValue, error) {
	resultByMetricArr, err := redis.Values(values, nil)
	if err != nil {
		if err == redis.ErrNil {
			return make([]*moira.MetricValue, 0), nil
		}
		return nil, fmt.Errorf("failed to read metricValues: %s", err.Error())
	}
	metricsValues := make([]*moira.MetricValue, 0, len(resultByMetricArr)/2)
	for i := 0; i < len(resultByMetricArr); i += 2 {
		val, err := redis.String(resultByMetricArr[i], nil)
		if err != nil {
			return nil, err
		}
		valuesArr := strings.Split(val, " ")
		if len(valuesArr) != 2 {
			return nil, fmt.Errorf("value format is not valid: %s", val)
		}
		timestamp, err := strconv.ParseInt(valuesArr[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("metric timestamp format is not valid: %s", err.Error())
		}
		value, err := strconv.ParseFloat(valuesArr[1], 64)
		if err != nil {
			return nil, fmt.Errorf("metric value format is not valid: %s", err.Error())
		}
		retentionTimestamp, err := redis.Int64(resultByMetricArr[i+1], nil)
		if err != nil {
			return nil, fmt.Errorf("retention timestamp format is not valid: %s", err.Error())
		}
		metricValue := moira.MetricValue{
			RetentionTimestamp: retentionTimestamp,
			Timestamp:          timestamp,
			Value:              value,
		}
		metricsValues = append(metricsValues, &metricValue)
	}
	return metricsValues, nil
}
