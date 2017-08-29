package redis

import "github.com/garyburd/redigo/redis"

// UpdateMetricsHeartbeat increments redis counter
func (connector *DbConnector) UpdateMetricsHeartbeat() error {
	c := connector.pool.Get()
	defer c.Close()
	err := c.Send("INCR", moiraSelfStateMetricsHeartbeat)
	return err
}

// GetMetricsCount return metrics count received by Moira-Cache
func (connector *DbConnector) GetMetricsCount() (int64, error) {
	c := connector.pool.Get()
	defer c.Close()
	ts, err := redis.Int64(c.Do("GET", moiraSelfStateMetricsHeartbeat))
	if err == redis.ErrNil {
		return 0, nil
	}
	return ts, err
}

// GetChecksCount return checks count by Moira-Checker
func (connector *DbConnector) GetChecksCount() (int64, error) {
	c := connector.pool.Get()
	defer c.Close()
	ts, err := redis.Int64(c.Do("GET", moiraSelfStateChecksCounter))
	if err == redis.ErrNil {
		return 0, nil
	}
	return ts, err
}

var moiraSelfStateMetricsHeartbeat = "moira-selfstate:metrics-heartbeat"
var moiraSelfStateChecksCounter = "moira-selfstate:checks-counter"
