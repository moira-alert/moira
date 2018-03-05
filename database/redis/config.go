package redis

// Config - Redis database connection config
type Config struct {
	MasterName        string
	SentinelAddresses []string
	Host              string
	Port              string
	DBID              int
}
