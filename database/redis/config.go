package redis

import "time"

type ClusteringMode string

const (
	ClusteringModeStandalone ClusteringMode = "standalone"
	ClusteringModeSentinel   ClusteringMode = "sentinel"
	ClusteringModeCluster    ClusteringMode = "cluster"
)

// Config - Redis database connection config
type Config struct {
	RedisMode         ClusteringMode
	ClusterAddrs      []string
	MasterName        string
	SentinelAddresses []string
	Host              string
	Port              string
	DB                int
	ConnectionLimit   int
	AllowSlaveReads   bool
	MetricsTTL        time.Duration
}
