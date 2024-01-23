package redis

import (
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

func (connector *DbConnector) AddTriggersToCheck(clusterKey moira.ClusterKey, triggerIDs []string) error {
	key, err := makeTriggersToCheckKey(clusterKey)
	if err != nil {
		return err
	}
	return connector.addTriggersToCheck(key, triggerIDs)
}

func (connector *DbConnector) GetTriggersToCheck(clusterKey moira.ClusterKey, count int) ([]string, error) {
	key, err := makeTriggersToCheckKey(clusterKey)
	if err != nil {
		return nil, err
	}
	return connector.getTriggersToCheck(key, count)
}

func (connector *DbConnector) GetTriggersToCheckCount(clusterKey moira.ClusterKey) (int64, error) {
	key, err := makeTriggersToCheckKey(clusterKey)
	if err != nil {
		return 0, err
	}
	return connector.getTriggersToCheckCount(key)
}

// AddLocalTriggersToCheck gets trigger IDs and save it to Redis Set
func (connector *DbConnector) AddLocalTriggersToCheck(triggerIDs []string) error {
	return connector.addTriggersToCheck(localTriggersToCheckKey, triggerIDs)
}

// AddRemoteTriggersToCheck gets remote trigger IDs and save it to Redis Set
func (connector *DbConnector) AddRemoteTriggersToCheck(triggerIDs []string) error {
	return connector.addTriggersToCheck(remoteTriggersToCheckKey, triggerIDs)
}

func (connector *DbConnector) AddPrometheusTriggersToCheck(triggerIDs []string) error {
	return connector.addTriggersToCheck(prometheusTriggersToCheckKey, triggerIDs)
}

// GetLocalTriggersToCheck return random trigger ID from Redis Set
func (connector *DbConnector) GetLocalTriggersToCheck(count int) ([]string, error) {
	return connector.getTriggersToCheck(localTriggersToCheckKey, count)
}

// GetRemoteTriggersToCheck return random remote trigger ID from Redis Set
func (connector *DbConnector) GetRemoteTriggersToCheck(count int) ([]string, error) {
	return connector.getTriggersToCheck(remoteTriggersToCheckKey, count)
}

func (connector *DbConnector) GetPrometheusTriggersToCheck(count int) ([]string, error) {
	return connector.getTriggersToCheck(prometheusTriggersToCheckKey, count)
}

// GetLocalTriggersToCheckCount return number of triggers ID to check from Redis Set
func (connector *DbConnector) GetLocalTriggersToCheckCount() (int64, error) {
	return connector.getTriggersToCheckCount(localTriggersToCheckKey)
}

// GetRemoteTriggersToCheckCount return number of remote triggers ID to check from Redis Set
func (connector *DbConnector) GetRemoteTriggersToCheckCount() (int64, error) {
	return connector.getTriggersToCheckCount(remoteTriggersToCheckKey)
}

func (connector *DbConnector) GetPrometheusTriggersToCheckCount() (int64, error) {
	return connector.getTriggersToCheckCount(prometheusTriggersToCheckKey)
}

func (connector *DbConnector) addTriggersToCheck(key string, triggerIDs []string) error {
	ctx := connector.context
	pipe := (*connector.client).TxPipeline()

	for _, triggerID := range triggerIDs {
		pipe.SAdd(ctx, key, triggerID)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add triggers to check: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) getTriggersToCheck(key string, count int) ([]string, error) {
	ctx := connector.context
	c := *connector.client

	triggerIDs, err := c.SPopN(ctx, key, int64(count)).Result()
	if err != nil {
		if err == redis.Nil {
			return make([]string, 0), database.ErrNil
		}
		return make([]string, 0), fmt.Errorf("failed to pop trigger to check: %s", err.Error())
	}
	return triggerIDs, err
}

func (connector *DbConnector) getTriggersToCheckCount(key string) (int64, error) {
	ctx := connector.context
	c := *connector.client

	triggersToCheckCount, err := c.SCard(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get trigger to check count: %s", err.Error())
	}
	return triggersToCheckCount, nil
}

const (
	remoteTriggersToCheckKey     = "moira-remote-triggers-to-check"
	prometheusTriggersToCheckKey = "moira-prometheus-triggers-to-check"
	localTriggersToCheckKey      = "moira-triggers-to-check"
)

func makeTriggersToCheckKey(clusterKey moira.ClusterKey) (string, error) {
	var prefix string

	switch clusterKey.TriggerSource {
	case moira.GraphiteLocal:
		prefix = localTriggersToCheckKey

	case moira.GraphiteRemote:
		prefix = remoteTriggersToCheckKey

	case moira.PrometheusRemote:
		prefix = prometheusTriggersToCheckKey

	default:
		return "", fmt.Errorf("unknown trigger source `%s`", clusterKey.TriggerSource.String())
	}

	return makeTriggersToCheckKeyByClusterId(prefix, clusterKey.ClusterId), nil
}

func makeTriggersToCheckKeyByClusterId(prefix, clusterId string) string {
	if clusterId == "default" {
		return prefix
	}
	return fmt.Sprintf("%s:%s", prefix, clusterId)
}
