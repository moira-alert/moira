package goredis

// Config - Redis database connection config
type Config struct {
	MasterName string
	Addrs      []string
}
