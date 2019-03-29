package redis

import (
	"fmt"

	"github.com/garyburd/redigo/redis"
)

const (
	matchedMechanism = "matched"
)

// GetMatchedMetricsValues returns matched metrics values for given interval
func (connector *DbConnector) GetMatchedMetricsValues(from int64, until int64) (map[string][]int64, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("ZRANGEBYSCORE", protectorDataKey(matchedMechanism), from, until, "WITHSCORES")
	matchedValues, err := redis.Values(c.Do(""))
	if err != nil {
		return nil, fmt.Errorf("failed to get matched metric values: %s", err.Error())
	}

	res := make(map[string][]int64, len(matchedValues))
	return res, nil
}

// SaveMatchedMetricsCount saves matched metrics value
func (connector *DbConnector) SaveMatchedMetricsValue(source string, timestamp int64, value int64) error {
	c := connector.pool.Get()
	defer c.Close()
	countValue := fmt.Sprintf("%v %v", source, value)
	c.Send("ZADD", protectorDataKey(matchedMechanism), timestamp, countValue)
	return c.Flush()
}

// MatchedMetricsValues removes matched metrics values taken from 0 to given time
func (connector *DbConnector) RemoveMatchedMetricsValues(toTime int64) error {
	c := connector.pool.Get()
	defer c.Close()
	if _, err := c.Do("ZREMRANGEBYSCORE", protectorDataKey(matchedMechanism), "-inf", toTime); err != nil {
		return fmt.Errorf("failed to remove matched metrics values from -inf to %v, error: %v", toTime, err)
	}
	return nil
}

func protectorDataKey(mechanism string) string {
	return fmt.Sprintf("moira-protector-data:%s", mechanism)
}
