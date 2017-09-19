package redis

import "github.com/garyburd/redigo/redis"

// UpdateMetricsHeartbeat increments redis counter
func (connector *DbConnector) UpdateMetricsHeartbeat() error {
	c := connector.pool.Get()
	if c.Err() != nil {
		return c.Err()
	}
	defer c.Close()
	err := c.Send("INCR", selfStateMetricsHeartbeatKey)
	return err
}

// GetMetricsUpdatesCount return metrics count received by Moira-Filter
func (connector *DbConnector) GetMetricsUpdatesCount() (int64, error) {
	c := connector.pool.Get()
	if c.Err() != nil {
		return 0, c.Err()
	}
	defer c.Close()
	ts, err := redis.Int64(c.Do("GET", selfStateMetricsHeartbeatKey))
	if err == redis.ErrNil {
		return 0, nil
	}
	return ts, err
}

// GetChecksUpdatesCount return checks count by Moira-Checker
func (connector *DbConnector) GetChecksUpdatesCount() (int64, error) {
	c := connector.pool.Get()
	if c.Err() != nil {
		return 0, c.Err()
	}
	defer c.Close()
	ts, err := redis.Int64(c.Do("GET", selfStateChecksCounterKey))
	if err == redis.ErrNil {
		return 0, nil
	}
	return ts, err
}

var selfStateMetricsHeartbeatKey = "moira-selfstate:metrics-heartbeat"
var selfStateChecksCounterKey = "moira-selfstate:checks-counter"
