package graphite

//Config for graphite connection settings
type Config struct {
	URI      string
	Prefix   string
	Interval int64
}
