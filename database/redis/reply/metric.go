package reply

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
)

// MetricValues converts redis DB reply struct "RetentionTimestamp Value" "Timestamp" to moira.MetricValue object
func MetricValues(values *redis.ZSliceCmd) ([]*moira.MetricValue, error) {
	resultByMetricArr, err := values.Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return make([]*moira.MetricValue, 0), nil
		}
		return nil, fmt.Errorf("failed to read metricValues: %s", err.Error())
	}
	metricsValues := make([]*moira.MetricValue, 0, len(resultByMetricArr))
	for i := 0; i < len(resultByMetricArr); i++ {
		val := resultByMetricArr[i].Member.(string)
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
		retentionTimestamp := int64(resultByMetricArr[i].Score)
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
